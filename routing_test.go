package slackscot

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/metric"
	"log"
	"math"
	"os"
	"testing"
)

func TestNewPartitioner(t *testing.T) {
	tests := map[string]struct {
		partitionCount int
		expectedError  string
	}{
		"InvalidZeroPartitions": {
			partitionCount: 0,
			expectedError:  "A partition router can only work with a partitionCount that is a power of two but was [0]",
		},
		"ValidOnePartition": {
			partitionCount: 1,
			expectedError:  "",
		},
		"ValidTwoPartitions": {
			partitionCount: 2,
			expectedError:  "",
		},
		"Invalid3Partitions": {
			partitionCount: 3,
			expectedError:  "A partition router can only work with a partitionCount that is a power of two but was [3]",
		},
		"Valid4Partitions": {
			partitionCount: 4,
			expectedError:  "",
		},
		"Invalid5Partitions": {
			partitionCount: 5,
			expectedError:  "A partition router can only work with a partitionCount that is a power of two but was [5]",
		},
		"Valid8Partitions": {
			partitionCount: 8,
			expectedError:  "",
		},
		"Valid16Partitions": {
			partitionCount: 16,
			expectedError:  "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ins, _ := newInstrumenter("test", metric.Meter{}, func(ctx context.Context, result metric.Int64ObserverResult) {})
			pr, err := newPartitionRouter(tc.partitionCount, 1, nil, ins)

			if tc.expectedError == "" {
				assert.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Len(t, pr.messageQueues, tc.partitionCount)
				assert.Len(t, pr.workerTerminationSignals, tc.partitionCount)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestConsistentHashing(t *testing.T) {
	msgID := SlackMessageID{channelID: "general", timestamp: "11298321983.23"}

	for i := 0; i < 16; i++ {
		partitionCount := int(math.Pow(float64(2), float64(i)))
		name := fmt.Sprintf("With_%d_Partitions", partitionCount)

		t.Run(name, func(t *testing.T) {
			ins, _ := newInstrumenter("test", metric.Meter{}, func(ctx context.Context, result metric.Int64ObserverResult) {})
			pr, _ := newPartitionRouter(partitionCount, 1, &sLogger{logger: log.New(os.Stdout, "", log.LstdFlags)}, ins)
			partition := pr.partitionForMsgID(msgID)

			for i := 0; i < 100; i++ {
				assert.Equal(t, partition, pr.partitionForMsgID(msgID))
			}
		})
	}
}

func TestHashDistribution(t *testing.T) {
	// Generate message IDs that are all different to validate the uniform distribution across partitions
	msgIDs := make([]SlackMessageID, 0)
	for i := 0; i < 500000; i++ {
		msgTimestamp := fmt.Sprintf("19292929%d.214", i*1000)
		msgIDs = append(msgIDs, SlackMessageID{channelID: "general", timestamp: msgTimestamp})
		msgIDs = append(msgIDs, SlackMessageID{channelID: "général", timestamp: msgTimestamp})
	}

	msgIDCount := len(msgIDs)

	for i := 0; i < 10; i++ {
		partitionCount := int(math.Pow(float64(2), float64(i)))
		name := fmt.Sprintf("With_%d_Partitions", partitionCount)
		partitionHitCount := make([]int, partitionCount)

		t.Run(name, func(t *testing.T) {
			ins, _ := newInstrumenter("test", metric.Meter{}, func(ctx context.Context, result metric.Int64ObserverResult) {})
			pr, _ := newPartitionRouter(partitionCount, 1, &sLogger{logger: log.New(os.Stdout, "", log.LstdFlags)}, ins)

			for _, msgID := range msgIDs {
				partition := pr.partitionForMsgID(msgID)
				partitionHitCount[partition] = partitionHitCount[partition] + 1
			}

			expectedHitsPerPartition := float64(msgIDCount) / float64(partitionCount)
			deviationTolerance := 3.0 * expectedHitsPerPartition / 100
			for partition, hitCount := range partitionHitCount {
				assert.InDeltaf(t, expectedHitsPerPartition, hitCount, deviationTolerance, "All partitions should have received about [%.1f] hits but partition [%d] got [%d]", expectedHitsPerPartition, partition, hitCount)
			}
		})
	}
}

func TestHashMask(t *testing.T) {
	for i := 0; i < 16; i++ {
		partitionCount := int(math.Pow(float64(2), float64(i)))
		name := fmt.Sprintf("With_%d_Partitions", partitionCount)

		t.Run(name, func(t *testing.T) {
			mask := hashMask(partitionCount)
			assert.Equal(t, partitionCount-1, mask)
		})
	}
}
