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
	appName       string
	coreMetrics   coreMetrics
	pluginMetrics map[string]pluginMetrics
	meter         metric.Meter
}

type coreMetrics struct {
	msgsSeen                   metric.BoundInt64Counter
	msgsProcessed              map[string]metric.BoundInt64Counter
	msgProcessingLatencyMillis map[string]metric.BoundInt64Measure
	msgDispatchLatencyMillis   metric.BoundInt64Measure
	slackLatencyMillis         metric.BoundInt64Gauge
}

type pluginMetrics struct {
	processingTimeMillis metric.BoundInt64Measure
	reactionCount        metric.BoundInt64Counter
}

func newInstrumenter(appName string, meter metric.Meter) (ins *instrumenter) {
	ins = new(instrumenter)

	defaultLabels := meter.Labels(key.New("name").String(appName))

	msgSeen := meter.NewInt64Counter("msgSeen", metric.WithKeys(key.New("name")))
	slackLatency := meter.NewInt64Gauge("slackLatencyMillis", metric.WithKeys(key.New("name")))
	dispatchLatency := meter.NewInt64Measure("msgDispatchLatencyMillis", metric.WithKeys(key.New("name")))
	ins.coreMetrics = coreMetrics{msgsSeen: msgSeen.Bind(defaultLabels),
		msgsProcessed:              newBoundCounterByMsgType("msgProcessed", appName, meter),
		msgProcessingLatencyMillis: newBoundMeasureByMsgType("msgProcessingLatencyMillis", appName, meter),
		msgDispatchLatencyMillis:   dispatchLatency.Bind(defaultLabels),
		slackLatencyMillis:         slackLatency.Bind(defaultLabels)}

	ins.appName = appName
	ins.pluginMetrics = make(map[string]pluginMetrics)

	ins.meter = meter
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

func (ins *instrumenter) getOrCreatePluginMetrics(pluginName string) (pm pluginMetrics) {
	if pm, ok := ins.pluginMetrics[pluginName]; !ok {
		pm = newPluginMetrics(ins.appName, pluginName, ins.meter)
		ins.pluginMetrics[pluginName] = pm
	}

	return ins.pluginMetrics[pluginName]
}

func newPluginMetrics(appName string, pluginName string, meter metric.Meter) (pm pluginMetrics) {
	c := meter.NewInt64Counter("reactionCount", metric.WithKeys(key.New("name"), key.New("plugin")))
	m := meter.NewInt64Measure("processingTimeMillis", metric.WithKeys(key.New("name"), key.New("plugin")))

	pm.reactionCount = c.Bind(meter.Labels(key.New("name").String(appName), key.New("plugin").String(pluginName)))
	pm.processingTimeMillis = m.Bind(meter.Labels(key.New("name").String(appName), key.New("plugin").String(pluginName)))

	return pm
}

type timed func()

func measure(operation timed) (d time.Duration) {
	before := time.Now()

	operation()

	return time.Now().Sub(before)
}
