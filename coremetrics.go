package slackscot

import (
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"time"
)

const (
	newMsgType    = "new"
	updateMsgType = "edit"
	deleteMsgType = "delete"
)

// instrumenter holds data for core instrumentation
type instrumenter struct {
	appName       string
	coreMetrics   coreMetrics
	pluginMetrics map[string]pluginMetrics
	meter         metric.Meter
}

// coreMetrics holds core slackscot metrics
type coreMetrics struct {
	msgsSeen                   metric.BoundInt64Counter
	msgsProcessed              map[string]metric.BoundInt64Counter
	msgProcessingLatencyMillis map[string]metric.BoundInt64ValueRecorder
	msgDispatchLatencyMillis   metric.BoundInt64ValueRecorder
	slackLatencyMillis         metric.Int64ValueObserver
}

// pluginMetrics holds metrics specific to a plugin
type pluginMetrics struct {
	processingTimeMillis metric.BoundInt64ValueRecorder
	reactionCount        metric.BoundInt64Counter
}

// newInstrumenter creates a new core instrumenter
func newInstrumenter(appName string, meter metric.Meter, latencyCallback metric.Int64ObserverCallback) (ins *instrumenter, err error) {
	ins = new(instrumenter)

	defaultLabels := []kv.KeyValue{kv.Key("name").String(appName)}

	msgSeen, err := meter.NewInt64Counter("msgSeen")
	if err != nil {
		return nil, err
	}

	slackLatency, err := meter.NewInt64ValueObserver("slackLatencyMillis", latencyCallback)
	if err != nil {
		return nil, err
	}

	dispatchLatency, err := meter.NewInt64ValueRecorder("msgDispatchLatencyMillis")
	if err != nil {
		return nil, err
	}

	msgProcessed, err := newBoundCounterByMsgType("msgProcessed", appName, meter)
	if err != nil {
		return nil, err
	}

	msgProcessingLatencyMillis, err := newBoundValueRecorderByMsgType("msgProcessingLatencyMillis", appName, meter)
	if err != nil {
		return nil, err
	}

	ins.coreMetrics = coreMetrics{msgsSeen: msgSeen.Bind(defaultLabels...),
		msgsProcessed:              msgProcessed,
		msgProcessingLatencyMillis: msgProcessingLatencyMillis,
		msgDispatchLatencyMillis:   dispatchLatency.Bind(defaultLabels...),
		slackLatencyMillis:         slackLatency}

	ins.appName = appName
	ins.pluginMetrics = make(map[string]pluginMetrics)

	ins.meter = meter
	return ins, nil
}

// newBoundValueRecorderByMsgType creates a set of BoundInt64Counter by message type
func newBoundCounterByMsgType(counterName string, appName string, meter metric.Meter) (boundCounter map[string]metric.BoundInt64Counter, err error) {
	boundCounter = make(map[string]metric.BoundInt64Counter)

	c, err := meter.NewInt64Counter(counterName)
	if err != nil {
		return nil, err
	}

	boundCounter[newMsgType] = c.Bind(kv.Key("name").String(appName), kv.Key("msgType").String(newMsgType))
	boundCounter[updateMsgType] = c.Bind(kv.Key("name").String(appName), kv.Key("msgType").String(updateMsgType))
	boundCounter[deleteMsgType] = c.Bind(kv.Key("name").String(appName), kv.Key("msgType").String(deleteMsgType))

	return boundCounter, nil
}

// newBoundValueRecorderByMsgType creates a set of BoundInt64ValueRecorder by message type
func newBoundValueRecorderByMsgType(ValueRecorderName string, appName string, meter metric.Meter) (boundValueRecorder map[string]metric.BoundInt64ValueRecorder, err error) {
	boundValueRecorder = make(map[string]metric.BoundInt64ValueRecorder)

	m, err := meter.NewInt64ValueRecorder(ValueRecorderName)
	if err != nil {
		return nil, err
	}

	boundValueRecorder[newMsgType] = m.Bind(kv.Key("name").String(appName), kv.Key("msgType").String(newMsgType))
	boundValueRecorder[updateMsgType] = m.Bind(kv.Key("name").String(appName), kv.Key("msgType").String(updateMsgType))
	boundValueRecorder[deleteMsgType] = m.Bind(kv.Key("name").String(appName), kv.Key("msgType").String(deleteMsgType))

	return boundValueRecorder, nil
}

// getOrCreatePluginMetrics returns an existing pluginMetrics for a plugin or creates a new one, if necessary
func (ins *instrumenter) getOrCreatePluginMetrics(pluginName string) (pm pluginMetrics, err error) {
	if _, ok := ins.pluginMetrics[pluginName]; !ok {
		pm, err = newPluginMetrics(ins.appName, pluginName, ins.meter)
		if err != nil {
			return pm, err
		}
		ins.pluginMetrics[pluginName] = pm
	}

	return ins.pluginMetrics[pluginName], nil
}

// newPluginMetrics returns a new pluginMetrics instance for a plugin
func newPluginMetrics(appName string, pluginName string, meter metric.Meter) (pm pluginMetrics, err error) {
	c, err := meter.NewInt64Counter("reactionCount")
	if err != nil {
		return pm, err
	}
	m, err := meter.NewInt64ValueRecorder("processingTimeMillis")
	if err != nil {
		return pm, err
	}

	pm.reactionCount = c.Bind(kv.Key("name").String(appName), kv.Key("plugin").String(pluginName))
	pm.processingTimeMillis = m.Bind(kv.Key("name").String(appName), kv.Key("plugin").String(pluginName))

	return pm, nil
}

type timed func()

// measure returns the execution duration of a timed function
func measure(operation timed) (d time.Duration) {
	before := time.Now()

	operation()

	return time.Now().Sub(before)
}
