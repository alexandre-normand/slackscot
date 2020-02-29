package slackscot

import (
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"time"
)

const (
	newMsgType    = "new"
	updateMsgType = "edit"
	deleteMsgType = "delete"
)

type instrumenter struct {
	metrics CoreMetrics
}

type CoreMetrics struct {
	msgsSeen                   metric.BoundInt64Counter
	msgsProcessed              map[string]metric.BoundInt64Counter
	msgProcessingLatencyMillis map[string]metric.BoundInt64Measure
	msgDispatchLatencyMillis   metric.BoundInt64Measure
	slackLatencyMillis         metric.BoundInt64Gauge
}

func newInstrumenter(appName string, meter metric.Meter) (ins *instrumenter) {
	ins = new(instrumenter)

	defaultLabels := meter.Labels(key.New("name").String(appName))

	msgSeen := meter.NewInt64Counter("msgSeen", metric.WithKeys(key.New("name")))
	slackLatency := meter.NewInt64Gauge("slackLatencyMillis", metric.WithKeys(key.New("name")))
	dispatchLatency := meter.NewInt64Measure("msgDispatchLatencyMillis", metric.WithKeys(key.New("name")))
	ins.metrics = CoreMetrics{msgsSeen: msgSeen.Bind(defaultLabels),
		msgsProcessed:              newBoundCounterByMsgType("msgProcessed", appName, meter),
		msgProcessingLatencyMillis: newBoundMeasureByMsgType("msgProcessingLatencyMillis", appName, meter),
		msgDispatchLatencyMillis:   dispatchLatency.Bind(defaultLabels),
		slackLatencyMillis:         slackLatency.Bind(defaultLabels)}

	return ins
}

func newBoundCounterByMsgType(counterName string, appName string, meter metric.Meter) (boundCounter map[string]metric.BoundInt64Counter) {
	boundCounter = make(map[string]metric.BoundInt64Counter)

	c := meter.NewInt64Counter(counterName, metric.WithKeys(key.New("name"), key.New("msgType")))
	boundCounter[newMsgType] = c.Bind(meter.Labels(key.New("name").String(appName), key.New("msgType").String(newMsgType)))
	boundCounter[updateMsgType] = c.Bind(meter.Labels(key.New("name").String(appName), key.New("msgType").String(updateMsgType)))
	boundCounter[deleteMsgType] = c.Bind(meter.Labels(key.New("name").String(appName), key.New("msgType").String(deleteMsgType)))

	return boundCounter
}

func newBoundMeasureByMsgType(measureName string, appName string, meter metric.Meter) (boundMeasure map[string]metric.BoundInt64Measure) {
	boundMeasure = make(map[string]metric.BoundInt64Measure)

	m := meter.NewInt64Measure(measureName, metric.WithKeys(key.New("name"), key.New("msgType")))
	boundMeasure[newMsgType] = m.Bind(meter.Labels(key.New("name").String(appName), key.New("msgType").String(newMsgType)))
	boundMeasure[updateMsgType] = m.Bind(meter.Labels(key.New("name").String(appName), key.New("msgType").String(updateMsgType)))
	boundMeasure[deleteMsgType] = m.Bind(meter.Labels(key.New("name").String(appName), key.New("msgType").String(deleteMsgType)))

	return boundMeasure
}

type timed func()

func measure(operation timed) (d time.Duration) {
	before := time.Now()

	operation()

	return time.Now().Sub(before)
}
