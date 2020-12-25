package slackscot

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using opentelemetry.template template

//go:generate gowrap gen -p github.com/alexandre-normand/slackscot -i chatDriver -t opentelemetry.template -o chatdrivermetrics.go

import (
	"context"
	"time"
	"unicode"

	"github.com/slack-go/slack"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
)

// chatDriverWithTelemetry implements chatDriver interface with all methods wrapped
// with open telemetry metrics
type chatDriverWithTelemetry struct {
	base                     chatDriver
	methodCounters           map[string]metric.BoundInt64Counter
	errCounters              map[string]metric.BoundInt64Counter
	methodTimeValueRecorders map[string]metric.BoundInt64ValueRecorder
}

// NewchatDriverWithTelemetry returns an instance of the chatDriver decorated with open telemetry timing and count metrics
func NewchatDriverWithTelemetry(base chatDriver, name string, meter metric.Meter) chatDriverWithTelemetry {
	return chatDriverWithTelemetry{
		base:                     base,
		methodCounters:           newchatDriverMethodCounters("Calls", name, meter),
		errCounters:              newchatDriverMethodCounters("Errors", name, meter),
		methodTimeValueRecorders: newchatDriverMethodTimeValueRecorders(name, meter),
	}
}

func newchatDriverMethodTimeValueRecorders(appName string, meter metric.Meter) (boundTimeValueRecorders map[string]metric.BoundInt64ValueRecorder) {
	boundTimeValueRecorders = make(map[string]metric.BoundInt64ValueRecorder)
	mt := metric.Must(meter)

	nDeleteMessageValRecorder := []rune("chatDriver_DeleteMessage_ProcessingTimeMillis")
	nDeleteMessageValRecorder[0] = unicode.ToLower(nDeleteMessageValRecorder[0])
	mDeleteMessage := mt.NewInt64ValueRecorder(string(nDeleteMessageValRecorder))
	boundTimeValueRecorders["DeleteMessage"] = mDeleteMessage.Bind(label.String("name", appName))

	nSendMessageValRecorder := []rune("chatDriver_SendMessage_ProcessingTimeMillis")
	nSendMessageValRecorder[0] = unicode.ToLower(nSendMessageValRecorder[0])
	mSendMessage := mt.NewInt64ValueRecorder(string(nSendMessageValRecorder))
	boundTimeValueRecorders["SendMessage"] = mSendMessage.Bind(label.String("name", appName))

	nUpdateMessageValRecorder := []rune("chatDriver_UpdateMessage_ProcessingTimeMillis")
	nUpdateMessageValRecorder[0] = unicode.ToLower(nUpdateMessageValRecorder[0])
	mUpdateMessage := mt.NewInt64ValueRecorder(string(nUpdateMessageValRecorder))
	boundTimeValueRecorders["UpdateMessage"] = mUpdateMessage.Bind(label.String("name", appName))

	return boundTimeValueRecorders
}

func newchatDriverMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)
	mt := metric.Must(meter)

	nDeleteMessageCounter := []rune("chatDriver_DeleteMessage_" + suffix)
	nDeleteMessageCounter[0] = unicode.ToLower(nDeleteMessageCounter[0])
	cDeleteMessage := mt.NewInt64Counter(string(nDeleteMessageCounter))
	boundCounters["DeleteMessage"] = cDeleteMessage.Bind(label.String("name", appName))

	nSendMessageCounter := []rune("chatDriver_SendMessage_" + suffix)
	nSendMessageCounter[0] = unicode.ToLower(nSendMessageCounter[0])
	cSendMessage := mt.NewInt64Counter(string(nSendMessageCounter))
	boundCounters["SendMessage"] = cSendMessage.Bind(label.String("name", appName))

	nUpdateMessageCounter := []rune("chatDriver_UpdateMessage_" + suffix)
	nUpdateMessageCounter[0] = unicode.ToLower(nUpdateMessageCounter[0])
	cUpdateMessage := mt.NewInt64Counter(string(nUpdateMessageCounter))
	boundCounters["UpdateMessage"] = cUpdateMessage.Bind(label.String("name", appName))

	return boundCounters
}

// DeleteMessage implements chatDriver
func (_d chatDriverWithTelemetry) DeleteMessage(channelID string, timestamp string) (rChannelID string, rTimestamp string, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["DeleteMessage"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["DeleteMessage"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeValueRecorders["DeleteMessage"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.DeleteMessage(channelID, timestamp)
}

// SendMessage implements chatDriver
func (_d chatDriverWithTelemetry) SendMessage(channelID string, options ...slack.MsgOption) (rChannelID string, rTimestamp string, rText string, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["SendMessage"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["SendMessage"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeValueRecorders["SendMessage"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.SendMessage(channelID, options...)
}

// UpdateMessage implements chatDriver
func (_d chatDriverWithTelemetry) UpdateMessage(channelID string, timestamp string, options ...slack.MsgOption) (rChannelID string, rTimestamp string, rText string, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["UpdateMessage"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["UpdateMessage"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeValueRecorders["UpdateMessage"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.UpdateMessage(channelID, timestamp, options...)
}
