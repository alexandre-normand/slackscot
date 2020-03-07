package slackscot

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using opentelemetry.template template

//go:generate gowrap gen -p github.com/alexandre-normand/slackscot -i chatDriver -t opentelemetry.template -o chatdrivermetrics.go

import (
	"context"
	"time"
	"unicode"

	"github.com/nlopes/slack"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
)

// chatDriverWithTelemetry implements chatDriver interface with all methods wrapped
// with open telemetry metrics
type chatDriverWithTelemetry struct {
	base               chatDriver
	methodCounters     map[string]metric.BoundInt64Counter
	errCounters        map[string]metric.BoundInt64Counter
	methodTimeMeasures map[string]metric.BoundInt64Measure
}

// NewchatDriverWithTelemetry returns an instance of the chatDriver decorated with open telemetry timing and count metrics
func NewchatDriverWithTelemetry(base chatDriver, name string, meter metric.Meter) chatDriverWithTelemetry {
	return chatDriverWithTelemetry{
		base:               base,
		methodCounters:     newMethodCounters("Calls", name, meter),
		errCounters:        newMethodCounters("Errors", name, meter),
		methodTimeMeasures: newMethodTimeMeasures(name, meter),
	}
}

func newMethodTimeMeasures(appName string, meter metric.Meter) (boundTimeMeasures map[string]metric.BoundInt64Measure) {
	boundTimeMeasures = make(map[string]metric.BoundInt64Measure)

	nDeleteMessageMeasure := []rune("DeleteMessage" + "DurationMillis")
	nDeleteMessageMeasure[0] = unicode.ToLower(nDeleteMessageMeasure[0])
	mDeleteMessage := meter.NewInt64Measure(string(nDeleteMessageMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["DeleteMessage"] = mDeleteMessage.Bind(meter.Labels(key.New("name").String(appName)))

	nSendMessageMeasure := []rune("SendMessage" + "DurationMillis")
	nSendMessageMeasure[0] = unicode.ToLower(nSendMessageMeasure[0])
	mSendMessage := meter.NewInt64Measure(string(nSendMessageMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["SendMessage"] = mSendMessage.Bind(meter.Labels(key.New("name").String(appName)))

	nUpdateMessageMeasure := []rune("UpdateMessage" + "DurationMillis")
	nUpdateMessageMeasure[0] = unicode.ToLower(nUpdateMessageMeasure[0])
	mUpdateMessage := meter.NewInt64Measure(string(nUpdateMessageMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["UpdateMessage"] = mUpdateMessage.Bind(meter.Labels(key.New("name").String(appName)))

	return boundTimeMeasures
}

func newMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)

	nDeleteMessageCounter := []rune("DeleteMessage" + suffix)
	nDeleteMessageCounter[0] = unicode.ToLower(nDeleteMessageCounter[0])
	cDeleteMessage := meter.NewInt64Counter(string(nDeleteMessageCounter), metric.WithKeys(key.New("name")))
	boundCounters["DeleteMessage"] = cDeleteMessage.Bind(meter.Labels(key.New("name").String(appName)))

	nSendMessageCounter := []rune("SendMessage" + suffix)
	nSendMessageCounter[0] = unicode.ToLower(nSendMessageCounter[0])
	cSendMessage := meter.NewInt64Counter(string(nSendMessageCounter), metric.WithKeys(key.New("name")))
	boundCounters["SendMessage"] = cSendMessage.Bind(meter.Labels(key.New("name").String(appName)))

	nUpdateMessageCounter := []rune("UpdateMessage" + suffix)
	nUpdateMessageCounter[0] = unicode.ToLower(nUpdateMessageCounter[0])
	cUpdateMessage := meter.NewInt64Counter(string(nUpdateMessageCounter), metric.WithKeys(key.New("name")))
	boundCounters["UpdateMessage"] = cUpdateMessage.Bind(meter.Labels(key.New("name").String(appName)))

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

		methodTimeMeasure := _d.methodTimeMeasures["DeleteMessage"]
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

		methodTimeMeasure := _d.methodTimeMeasures["SendMessage"]
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

		methodTimeMeasure := _d.methodTimeMeasures["UpdateMessage"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.UpdateMessage(channelID, timestamp, options...)
}
