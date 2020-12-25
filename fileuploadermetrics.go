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
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
)

// FileUploaderWithTelemetry implements FileUploader interface with all methods wrapped
// with open telemetry metrics
type FileUploaderWithTelemetry struct {
	base                     FileUploader
	methodCounters           map[string]metric.BoundInt64Counter
	errCounters              map[string]metric.BoundInt64Counter
	methodTimeValueRecorders map[string]metric.BoundInt64ValueRecorder
}

// NewFileUploaderWithTelemetry returns an instance of the FileUploader decorated with open telemetry timing and count metrics
func NewFileUploaderWithTelemetry(base FileUploader, name string, meter metric.Meter) FileUploaderWithTelemetry {
	return FileUploaderWithTelemetry{
		base:                     base,
		methodCounters:           newFileUploaderMethodCounters("Calls", name, meter),
		errCounters:              newFileUploaderMethodCounters("Errors", name, meter),
		methodTimeValueRecorders: newFileUploaderMethodTimeValueRecorders(name, meter),
	}
}

func newFileUploaderMethodTimeValueRecorders(appName string, meter metric.Meter) (boundTimeValueRecorders map[string]metric.BoundInt64ValueRecorder) {
	boundTimeValueRecorders = make(map[string]metric.BoundInt64ValueRecorder)
	mt := metric.Must(meter)

	nUploadFileValRecorder := []rune("FileUploader_UploadFile_ProcessingTimeMillis")
	nUploadFileValRecorder[0] = unicode.ToLower(nUploadFileValRecorder[0])
	mUploadFile := mt.NewInt64ValueRecorder(string(nUploadFileValRecorder))
	boundTimeValueRecorders["UploadFile"] = mUploadFile.Bind(label.String("name", appName))

	return boundTimeValueRecorders
}

func newFileUploaderMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)
	mt := metric.Must(meter)

	nUploadFileCounter := []rune("FileUploader_UploadFile_" + suffix)
	nUploadFileCounter[0] = unicode.ToLower(nUploadFileCounter[0])
	cUploadFile := mt.NewInt64Counter(string(nUploadFileCounter))
	boundCounters["UploadFile"] = cUploadFile.Bind(label.String("name", appName))

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

		methodTimeMeasure := _d.methodTimeValueRecorders["UploadFile"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.UploadFile(params, options...)
}
