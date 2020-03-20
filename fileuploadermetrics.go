package slackscot

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using opentelemetry.template template

//go:generate gowrap gen -p github.com/alexandre-normand/slackscot -i FileUploader -t opentelemetry.template -o fileuploadermetrics.go

import (
	"context"
	"time"
	"unicode"

	"github.com/slack-go/slack"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
)

// FileUploaderWithTelemetry implements FileUploader interface with all methods wrapped
// with open telemetry metrics
type FileUploaderWithTelemetry struct {
	base               FileUploader
	methodCounters     map[string]metric.BoundInt64Counter
	errCounters        map[string]metric.BoundInt64Counter
	methodTimeMeasures map[string]metric.BoundInt64Measure
}

// NewFileUploaderWithTelemetry returns an instance of the FileUploader decorated with open telemetry timing and count metrics
func NewFileUploaderWithTelemetry(base FileUploader, name string, meter metric.Meter) FileUploaderWithTelemetry {
	return FileUploaderWithTelemetry{
		base:               base,
		methodCounters:     newFileUploaderMethodCounters("Calls", name, meter),
		errCounters:        newFileUploaderMethodCounters("Errors", name, meter),
		methodTimeMeasures: newFileUploaderMethodTimeMeasures(name, meter),
	}
}

func newFileUploaderMethodTimeMeasures(appName string, meter metric.Meter) (boundTimeMeasures map[string]metric.BoundInt64Measure) {
	boundTimeMeasures = make(map[string]metric.BoundInt64Measure)

	nUploadFileMeasure := []rune("FileUploader_UploadFile_ProcessingTimeMillis")
	nUploadFileMeasure[0] = unicode.ToLower(nUploadFileMeasure[0])
	mUploadFile := meter.NewInt64Measure(string(nUploadFileMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["UploadFile"] = mUploadFile.Bind(meter.Labels(key.New("name").String(appName)))

	return boundTimeMeasures
}

func newFileUploaderMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)

	nUploadFileCounter := []rune("FileUploader_UploadFile_" + suffix)
	nUploadFileCounter[0] = unicode.ToLower(nUploadFileCounter[0])
	cUploadFile := meter.NewInt64Counter(string(nUploadFileCounter), metric.WithKeys(key.New("name")))
	boundCounters["UploadFile"] = cUploadFile.Bind(meter.Labels(key.New("name").String(appName)))

	return boundCounters
}

// UploadFile implements FileUploader
func (_d FileUploaderWithTelemetry) UploadFile(params slack.FileUploadParameters, options ...UploadOption) (file *slack.File, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["UploadFile"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["UploadFile"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["UploadFile"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.UploadFile(params, options...)
}
