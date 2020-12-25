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
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
)

// datastorerWithTelemetry implements datastorer interface with all methods wrapped
// with open telemetry metrics
type datastorerWithTelemetry struct {
	base                     datastorer
	methodCounters           map[string]metric.BoundInt64Counter
	errCounters              map[string]metric.BoundInt64Counter
	methodTimeValueRecorders map[string]metric.BoundInt64ValueRecorder
}

// NewdatastorerWithTelemetry returns an instance of the datastorer decorated with open telemetry timing and count metrics
func NewdatastorerWithTelemetry(base datastorer, name string, meter metric.Meter) datastorerWithTelemetry {
	return datastorerWithTelemetry{
		base:                     base,
		methodCounters:           newdatastorerMethodCounters("Calls", name, meter),
		errCounters:              newdatastorerMethodCounters("Errors", name, meter),
		methodTimeValueRecorders: newdatastorerMethodTimeValueRecorders(name, meter),
	}
}

func newdatastorerMethodTimeValueRecorders(appName string, meter metric.Meter) (boundTimeValueRecorders map[string]metric.BoundInt64ValueRecorder) {
	boundTimeValueRecorders = make(map[string]metric.BoundInt64ValueRecorder)
	mt := metric.Must(meter)

	nCloseValRecorder := []rune("datastorer_Close_ProcessingTimeMillis")
	nCloseValRecorder[0] = unicode.ToLower(nCloseValRecorder[0])
	mClose := mt.NewInt64ValueRecorder(string(nCloseValRecorder))
	boundTimeValueRecorders["Close"] = mClose.Bind(label.String("name", appName))

	nDeleteValRecorder := []rune("datastorer_Delete_ProcessingTimeMillis")
	nDeleteValRecorder[0] = unicode.ToLower(nDeleteValRecorder[0])
	mDelete := mt.NewInt64ValueRecorder(string(nDeleteValRecorder))
	boundTimeValueRecorders["Delete"] = mDelete.Bind(label.String("name", appName))

	nGetValRecorder := []rune("datastorer_Get_ProcessingTimeMillis")
	nGetValRecorder[0] = unicode.ToLower(nGetValRecorder[0])
	mGet := mt.NewInt64ValueRecorder(string(nGetValRecorder))
	boundTimeValueRecorders["Get"] = mGet.Bind(label.String("name", appName))

	nGetAllValRecorder := []rune("datastorer_GetAll_ProcessingTimeMillis")
	nGetAllValRecorder[0] = unicode.ToLower(nGetAllValRecorder[0])
	mGetAll := mt.NewInt64ValueRecorder(string(nGetAllValRecorder))
	boundTimeValueRecorders["GetAll"] = mGetAll.Bind(label.String("name", appName))

	nPutValRecorder := []rune("datastorer_Put_ProcessingTimeMillis")
	nPutValRecorder[0] = unicode.ToLower(nPutValRecorder[0])
	mPut := mt.NewInt64ValueRecorder(string(nPutValRecorder))
	boundTimeValueRecorders["Put"] = mPut.Bind(label.String("name", appName))

	nconnectValRecorder := []rune("datastorer_connect_ProcessingTimeMillis")
	nconnectValRecorder[0] = unicode.ToLower(nconnectValRecorder[0])
	mconnect := mt.NewInt64ValueRecorder(string(nconnectValRecorder))
	boundTimeValueRecorders["connect"] = mconnect.Bind(label.String("name", appName))

	return boundTimeValueRecorders
}

func newdatastorerMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)
	mt := metric.Must(meter)

	nCloseCounter := []rune("datastorer_Close_" + suffix)
	nCloseCounter[0] = unicode.ToLower(nCloseCounter[0])
	cClose := mt.NewInt64Counter(string(nCloseCounter))
	boundCounters["Close"] = cClose.Bind(label.String("name", appName))

	nDeleteCounter := []rune("datastorer_Delete_" + suffix)
	nDeleteCounter[0] = unicode.ToLower(nDeleteCounter[0])
	cDelete := mt.NewInt64Counter(string(nDeleteCounter))
	boundCounters["Delete"] = cDelete.Bind(label.String("name", appName))

	nGetCounter := []rune("datastorer_Get_" + suffix)
	nGetCounter[0] = unicode.ToLower(nGetCounter[0])
	cGet := mt.NewInt64Counter(string(nGetCounter))
	boundCounters["Get"] = cGet.Bind(label.String("name", appName))

	nGetAllCounter := []rune("datastorer_GetAll_" + suffix)
	nGetAllCounter[0] = unicode.ToLower(nGetAllCounter[0])
	cGetAll := mt.NewInt64Counter(string(nGetAllCounter))
	boundCounters["GetAll"] = cGetAll.Bind(label.String("name", appName))

	nPutCounter := []rune("datastorer_Put_" + suffix)
	nPutCounter[0] = unicode.ToLower(nPutCounter[0])
	cPut := mt.NewInt64Counter(string(nPutCounter))
	boundCounters["Put"] = cPut.Bind(label.String("name", appName))

	nconnectCounter := []rune("datastorer_connect_" + suffix)
	nconnectCounter[0] = unicode.ToLower(nconnectCounter[0])
	cconnect := mt.NewInt64Counter(string(nconnectCounter))
	boundCounters["connect"] = cconnect.Bind(label.String("name", appName))

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

		methodTimeMeasure := _d.methodTimeValueRecorders["Close"]
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

		methodTimeMeasure := _d.methodTimeValueRecorders["Delete"]
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

		methodTimeMeasure := _d.methodTimeValueRecorders["Get"]
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

		methodTimeMeasure := _d.methodTimeValueRecorders["GetAll"]
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

		methodTimeMeasure := _d.methodTimeValueRecorders["Put"]
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

		methodTimeMeasure := _d.methodTimeValueRecorders["connect"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.connect()
}
