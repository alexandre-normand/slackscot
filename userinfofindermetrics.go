package slackscot

// DO NOT EDIT!
// This code is generated with http://github.com/hexdigest/gowrap tool
// using opentelemetry.template template

//go:generate gowrap gen -p github.com/alexandre-normand/slackscot -i UserInfoFinder -t opentelemetry.template -o userinfofindermetrics.go

import (
	"context"
	"time"
	"unicode"

	"github.com/slack-go/slack"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
)

// UserInfoFinderWithTelemetry implements UserInfoFinder interface with all methods wrapped
// with open telemetry metrics
type UserInfoFinderWithTelemetry struct {
	base               UserInfoFinder
	methodCounters     map[string]metric.BoundInt64Counter
	errCounters        map[string]metric.BoundInt64Counter
	methodTimeMeasures map[string]metric.BoundInt64Measure
}

// NewUserInfoFinderWithTelemetry returns an instance of the UserInfoFinder decorated with open telemetry timing and count metrics
func NewUserInfoFinderWithTelemetry(base UserInfoFinder, name string, meter metric.Meter) UserInfoFinderWithTelemetry {
	return UserInfoFinderWithTelemetry{
		base:               base,
		methodCounters:     newUserInfoFinderMethodCounters("Calls", name, meter),
		errCounters:        newUserInfoFinderMethodCounters("Errors", name, meter),
		methodTimeMeasures: newUserInfoFinderMethodTimeMeasures(name, meter),
	}
}

func newUserInfoFinderMethodTimeMeasures(appName string, meter metric.Meter) (boundTimeMeasures map[string]metric.BoundInt64Measure) {
	boundTimeMeasures = make(map[string]metric.BoundInt64Measure)

	nGetUserInfoMeasure := []rune("UserInfoFinder_GetUserInfo_ProcessingTimeMillis")
	nGetUserInfoMeasure[0] = unicode.ToLower(nGetUserInfoMeasure[0])
	mGetUserInfo := meter.NewInt64Measure(string(nGetUserInfoMeasure), metric.WithKeys(key.New("name")))
	boundTimeMeasures["GetUserInfo"] = mGetUserInfo.Bind(meter.Labels(key.New("name").String(appName)))

	return boundTimeMeasures
}

func newUserInfoFinderMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)

	nGetUserInfoCounter := []rune("UserInfoFinder_GetUserInfo_" + suffix)
	nGetUserInfoCounter[0] = unicode.ToLower(nGetUserInfoCounter[0])
	cGetUserInfo := meter.NewInt64Counter(string(nGetUserInfoCounter), metric.WithKeys(key.New("name")))
	boundCounters["GetUserInfo"] = cGetUserInfo.Bind(meter.Labels(key.New("name").String(appName)))

	return boundCounters
}

// GetUserInfo implements UserInfoFinder
func (_d UserInfoFinderWithTelemetry) GetUserInfo(userID string) (user *slack.User, err error) {
	_since := time.Now()
	defer func() {
		if err != nil {
			errCounter := _d.errCounters["GetUserInfo"]
			errCounter.Add(context.Background(), 1)
		}

		methodCounter := _d.methodCounters["GetUserInfo"]
		methodCounter.Add(context.Background(), 1)

		methodTimeMeasure := _d.methodTimeMeasures["GetUserInfo"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.GetUserInfo(userID)
}
