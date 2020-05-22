package datastoredb

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using ../../opentelemetry.template template

//go:generate gowrap gen -p github.com/alexandre-normand/slackscot/store/datastoredb -i datastorer -t ../../opentelemetry.template -o datastorermetrics.go

import (
	"context"
	"time"
	"unicode"

	"cloud.google.com/go/datastore"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
)

// datastorerWithTelemetry implements datastorer interface with all methods wrapped
// with open telemetry metrics
type datastorerWithTelemetry struct {
	base               datastorer
	methodCounters     map[string]metric.BoundInt64Counter
	errCounters        map[string]metric.BoundInt64Counter
	methodTimeMeasures map[string]metric.BoundInt64Measure
}

// NewdatastorerWithTelemetry returns an instance of the datastorer decorated with open telemetry timing and count metrics
func NewdatastorerWithTelemetry(base datastorer, name string, meter metric.Meter) datastorerWithTelemetry {
	return datastorerWithTelemetry{
		base:               base,
		methodCounters:     newdatastorerMethodCounters("Calls", name, meter),
		errCounters:        newdatastorerMethodCounters("Errors", name, meter),
		methodTimeMeasures: newdatastorerMethodTimeMeasures(name, meter),
	}
}

func newdatastorerMethodTimeMeasures(appName string, meter metric.Meter) (boundTimeMeasures map[string]metric.BoundInt64Measure) {
	boundTimeMeasures = make(map[string]metric.BoundInt64Measure)

	nCloseMeasure := []rune("datastorer_Close_ProcessingTimeMillis")
	nCloseMeasure[0] = unicode.ToLower(nCloseMeasure[0])
	mClose := meter.NewInt64Measure(string(nCloseMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["Close"] = mClose.Bind(meter.Labels(key.New("name").String(appName)))

	nDeleteMeasure := []rune("datastorer_Delete_ProcessingTimeMillis")
	nDeleteMeasure[0] = unicode.ToLower(nDeleteMeasure[0])
	mDelete := meter.NewInt64Measure(string(nDeleteMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["Delete"] = mDelete.Bind(meter.Labels(key.New("name").String(appName)))

	nGetMeasure := []rune("datastorer_Get_ProcessingTimeMillis")
	nGetMeasure[0] = unicode.ToLower(nGetMeasure[0])
	mGet := meter.NewInt64Measure(string(nGetMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["Get"] = mGet.Bind(meter.Labels(key.New("name").String(appName)))

	nGetAllMeasure := []rune("datastorer_GetAll_ProcessingTimeMillis")
	nGetAllMeasure[0] = unicode.ToLower(nGetAllMeasure[0])
	mGetAll := meter.NewInt64Measure(string(nGetAllMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["GetAll"] = mGetAll.Bind(meter.Labels(key.New("name").String(appName)))

	nPutMeasure := []rune("datastorer_Put_ProcessingTimeMillis")
	nPutMeasure[0] = unicode.ToLower(nPutMeasure[0])
	mPut := meter.NewInt64Measure(string(nPutMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["Put"] = mPut.Bind(meter.Labels(key.New("name").String(appName)))

	nconnectMeasure := []rune("datastorer_connect_ProcessingTimeMillis")
	nconnectMeasure[0] = unicode.ToLower(nconnectMeasure[0])
	mconnect := meter.NewInt64Measure(string(nconnectMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["connect"] = mconnect.Bind(meter.Labels(key.New("name").String(appName)))

	return boundTimeMeasures
}

func newdatastorerMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)

	nCloseCounter := []rune("datastorer_Close_" + suffix)
	nCloseCounter[0] = unicode.ToLower(nCloseCounter[0])
	cClose := meter.NewInt64Counter(string(nCloseCounter), metric.WithKeys(key.New("name")))
	boundCounters["Close"] = cClose.Bind(meter.Labels(key.New("name").String(appName)))

	nDeleteCounter := []rune("datastorer_Delete_" + suffix)
	nDeleteCounter[0] = unicode.ToLower(nDeleteCounter[0])
	cDelete := meter.NewInt64Counter(string(nDeleteCounter), metric.WithKeys(key.New("name")))
	boundCounters["Delete"] = cDelete.Bind(meter.Labels(key.New("name").String(appName)))

	nGetCounter := []rune("datastorer_Get_" + suffix)
	nGetCounter[0] = unicode.ToLower(nGetCounter[0])
	cGet := meter.NewInt64Counter(string(nGetCounter), metric.WithKeys(key.New("name")))
	boundCounters["Get"] = cGet.Bind(meter.Labels(key.New("name").String(appName)))

	nGetAllCounter := []rune("datastorer_GetAll_" + suffix)
	nGetAllCounter[0] = unicode.ToLower(nGetAllCounter[0])
	cGetAll := meter.NewInt64Counter(string(nGetAllCounter), metric.WithKeys(key.New("name")))
	boundCounters["GetAll"] = cGetAll.Bind(meter.Labels(key.New("name").String(appName)))

	nPutCounter := []rune("datastorer_Put_" + suffix)
	nPutCounter[0] = unicode.ToLower(nPutCounter[0])
	cPut := meter.NewInt64Counter(string(nPutCounter), metric.WithKeys(key.New("name")))
	boundCounters["Put"] = cPut.Bind(meter.Labels(key.New("name").String(appName)))

	nconnectCounter := []rune("datastorer_connect_" + suffix)
	nconnectCounter[0] = unicode.ToLower(nconnectCounter[0])
	cconnect := meter.NewInt64Counter(string(nconnectCounter), metric.WithKeys(key.New("name")))
	boundCounters["connect"] = cconnect.Bind(meter.Labels(key.New("name").String(appName)))

	return boundCounters
}

// Close implements datastorer
func (_d datastorerWithTelemetry) Close() (err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["Close"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["Close"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["Close"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.Close()
}

// Delete implements datastorer
func (_d datastorerWithTelemetry) Delete(ctx context.Context, k *datastore.Key) (err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["Delete"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["Delete"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["Delete"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.Delete(ctx, k)
}

// Get implements datastorer
func (_d datastorerWithTelemetry) Get(ctx context.Context, k *datastore.Key, dest interface{}) (err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["Get"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["Get"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["Get"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.Get(ctx, k, dest)
}

// GetAll implements datastorer
func (_d datastorerWithTelemetry) GetAll(ctx context.Context, query *datastore.Query, dest interface{}) (keys []*datastore.Key, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["GetAll"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["GetAll"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["GetAll"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.GetAll(ctx, query, dest)
}

// Put implements datastorer
func (_d datastorerWithTelemetry) Put(ctx context.Context, k *datastore.Key, v interface{}) (key *datastore.Key, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["Put"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["Put"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["Put"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.Put(ctx, k, v)
}

// connect implements datastorer
func (_d datastorerWithTelemetry) connect() (err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["connect"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["connect"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["connect"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.connect()
}
