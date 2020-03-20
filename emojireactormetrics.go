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
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
)

// EmojiReactorWithTelemetry implements EmojiReactor interface with all methods wrapped
// with open telemetry metrics
type EmojiReactorWithTelemetry struct {
	base               EmojiReactor
	methodCounters     map[string]metric.BoundInt64Counter
	errCounters        map[string]metric.BoundInt64Counter
	methodTimeMeasures map[string]metric.BoundInt64Measure
}

// NewEmojiReactorWithTelemetry returns an instance of the EmojiReactor decorated with open telemetry timing and count metrics
func NewEmojiReactorWithTelemetry(base EmojiReactor, name string, meter metric.Meter) EmojiReactorWithTelemetry {
	return EmojiReactorWithTelemetry{
		base:               base,
		methodCounters:     newEmojiReactorMethodCounters("Calls", name, meter),
		errCounters:        newEmojiReactorMethodCounters("Errors", name, meter),
		methodTimeMeasures: newEmojiReactorMethodTimeMeasures(name, meter),
	}
}

func newEmojiReactorMethodTimeMeasures(appName string, meter metric.Meter) (boundTimeMeasures map[string]metric.BoundInt64Measure) {
	boundTimeMeasures = make(map[string]metric.BoundInt64Measure)

	nAddReactionMeasure := []rune("EmojiReactor_AddReaction_ProcessingTimeMillis")
	nAddReactionMeasure[0] = unicode.ToLower(nAddReactionMeasure[0])
	mAddReaction := meter.NewInt64Measure(string(nAddReactionMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["AddReaction"] = mAddReaction.Bind(meter.Labels(key.New("name").String(appName)))

	return boundTimeMeasures
}

func newEmojiReactorMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)

	nAddReactionCounter := []rune("EmojiReactor_AddReaction_" + suffix)
	nAddReactionCounter[0] = unicode.ToLower(nAddReactionCounter[0])
	cAddReaction := meter.NewInt64Counter(string(nAddReactionCounter), metric.WithKeys(key.New("name")))
	boundCounters["AddReaction"] = cAddReaction.Bind(meter.Labels(key.New("name").String(appName)))

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

		methodTimeMeasure := _d.methodTimeMeasures["AddReaction"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.AddReaction(name, item)
}
