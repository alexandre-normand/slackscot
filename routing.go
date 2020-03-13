package slackscot

import (
	"context"
	"fmt"
	"github.com/nlopes/slack"
	"hash"
	"hash/crc32"
	"math"
)

type partitionRouter struct {
	// Logger
	log *sLogger

	// messageQueues with partition keyed by the hash of the incoming message id
	// so that processing of messages (new, updates and deletes) are handled by
	// the same work queue therefore ensuring correct ordered processing
	// of those events
	messageQueues []chan slack.MessageEvent

	// workerTerminationSignals are channels receiving a termination signal for each
	// workerQueue
	workerTerminationSignals []chan bool

	// hash function to direct message processing to partitions
	hasher   hash.Hash32
	hashMask int

	*instrumenter
}

func newPartitionRouter(partitionCount int, queueBufferSize int, log *sLogger, instrumenter *instrumenter) (pr *partitionRouter, err error) {
	if !isPowerOfTwo(partitionCount) {
		return nil, fmt.Errorf("A partition router can only work with a partitionCount that is a power of two but was [%d]", partitionCount)
	}

	pr = new(partitionRouter)
	pr.messageQueues = make([]chan slack.MessageEvent, partitionCount)
	for i := range pr.messageQueues {
		pr.messageQueues[i] = make(chan slack.MessageEvent, queueBufferSize)
	}
	pr.workerTerminationSignals = make([]chan bool, partitionCount)
	for i := range pr.workerTerminationSignals {
		pr.workerTerminationSignals[i] = make(chan bool)
	}
	pr.hasher = crc32.NewIEEE()
	pr.hashMask = hashMask(partitionCount)
	pr.log = log
	pr.instrumenter = instrumenter

	return pr, nil
}

// routeMessageEvent routes the message processing to the correct partition based on its original message id to ensure
// that all message and its updates are processed in order
func (pr *partitionRouter) routeMessageEvent(msgEvent slack.MessageEvent) {
	msgID := getOriginalMessageID(msgEvent)

	partition := pr.partitionForMsgID(msgID)

	pr.log.Debugf("Dispatching message [%s] to partition [%d]", msgID, partition)
	d := measure(func() {
		pr.messageQueues[partition] <- msgEvent
	})

	pr.coreMetrics.msgDispatchLatencyMillis.Record(context.Background(), d.Milliseconds())
}

// partitionForMsgID returns the partition index for a given message ID
func (pr *partitionRouter) partitionForMsgID(msgID SlackMessageID) (partition int) {
	pr.hasher.Reset()
	pr.hasher.Write([]byte(msgID.channelID))
	pr.hasher.Write([]byte(msgID.timestamp))
	res := pr.hasher.Sum32()

	pr.log.Debugf("Hash [%d] calculated for [%s]", res, msgID)

	// Keep only the rightmost bits so we have a max equal to the partition count
	return int(res) & pr.hashMask
}

// isPowerOfTwo returns true if val is a power of two or false if not
func isPowerOfTwo(val int) bool {
	return (val != 0) && (val&(val-1)) == 0
}

// hashMask builds a mask for a partitionCount (which should be a power of two) to get a hash value
// that is in the range of the number of partitions we have
func hashMask(partitionCount int) int {
	maskSize := int(math.Log2(float64(partitionCount)))
	mask := 0
	for i := 0; i < maskSize; i++ {
		mask = mask<<1 | 1
	}

	return mask
}
