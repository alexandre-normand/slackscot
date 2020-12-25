package slackscot

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using opentelemetry.template template

//go:generate gowrap gen -p github.com/alexandre-normand/slackscot -i EmojiReactor -t opentelemetry.template -o emojireactormetrics.go

import (
	"context"
	"time"
	"unicode"

	"github.com/slack-go/slack"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
)

// EmojiReactorWithTelemetry implements EmojiReactor interface with all methods wrapped
// with open telemetry metrics
type EmojiReactorWithTelemetry struct {
	base                     EmojiReactor
	methodCounters           map[string]metric.BoundInt64Counter
	errCounters              map[string]metric.BoundInt64Counter
	methodTimeValueRecorders map[string]metric.BoundInt64ValueRecorder
}

// NewEmojiReactorWithTelemetry returns an instance of the EmojiReactor decorated with open telemetry timing and count metrics
func NewEmojiReactorWithTelemetry(base EmojiReactor, name string, meter metric.Meter) EmojiReactorWithTelemetry {
	return EmojiReactorWithTelemetry{
		base:                     base,
		methodCounters:           newEmojiReactorMethodCounters("Calls", name, meter),
		errCounters:              newEmojiReactorMethodCounters("Errors", name, meter),
		methodTimeValueRecorders: newEmojiReactorMethodTimeValueRecorders(name, meter),
	}
}

func newEmojiReactorMethodTimeValueRecorders(appName string, meter metric.Meter) (boundTimeValueRecorders map[string]metric.BoundInt64ValueRecorder) {
	boundTimeValueRecorders = make(map[string]metric.BoundInt64ValueRecorder)
	mt := metric.Must(meter)

	nAddReactionValRecorder := []rune("EmojiReactor_AddReaction_ProcessingTimeMillis")
	nAddReactionValRecorder[0] = unicode.ToLower(nAddReactionValRecorder[0])
	mAddReaction := mt.NewInt64ValueRecorder(string(nAddReactionValRecorder))
	boundTimeValueRecorders["AddReaction"] = mAddReaction.Bind(label.String("name", appName))

	return boundTimeValueRecorders
}

func newEmojiReactorMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)
	mt := metric.Must(meter)

	nAddReactionCounter := []rune("EmojiReactor_AddReaction_" + suffix)
	nAddReactionCounter[0] = unicode.ToLower(nAddReactionCounter[0])
	cAddReaction := mt.NewInt64Counter(string(nAddReactionCounter))
	boundCounters["AddReaction"] = cAddReaction.Bind(label.String("name", appName))

	return boundCounters
}

// AddReaction implements EmojiReactor
func (_d EmojiReactorWithTelemetry) AddReaction(name string, item slack.ItemRef) (err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["AddReaction"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["AddReaction"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeValueRecorders["AddReaction"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.AddReaction(name, item)
}
