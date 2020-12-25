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
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/metric"
)

// UserInfoFinderWithTelemetry implements UserInfoFinder interface with all methods wrapped
// with open telemetry metrics
type UserInfoFinderWithTelemetry struct {
	base                     UserInfoFinder
	methodCounters           map[string]metric.BoundInt64Counter
	errCounters              map[string]metric.BoundInt64Counter
	methodTimeValueRecorders map[string]metric.BoundInt64ValueRecorder
}

// NewUserInfoFinderWithTelemetry returns an instance of the UserInfoFinder decorated with open telemetry timing and count metrics
func NewUserInfoFinderWithTelemetry(base UserInfoFinder, name string, meter metric.Meter) UserInfoFinderWithTelemetry {
	return UserInfoFinderWithTelemetry{
		base:                     base,
		methodCounters:           newUserInfoFinderMethodCounters("Calls", name, meter),
		errCounters:              newUserInfoFinderMethodCounters("Errors", name, meter),
		methodTimeValueRecorders: newUserInfoFinderMethodTimeValueRecorders(name, meter),
	}
}

func newUserInfoFinderMethodTimeValueRecorders(appName string, meter metric.Meter) (boundTimeValueRecorders map[string]metric.BoundInt64ValueRecorder) {
	boundTimeValueRecorders = make(map[string]metric.BoundInt64ValueRecorder)
	mt := metric.Must(meter)

	nGetUserInfoValRecorder := []rune("UserInfoFinder_GetUserInfo_ProcessingTimeMillis")
	nGetUserInfoValRecorder[0] = unicode.ToLower(nGetUserInfoValRecorder[0])
	mGetUserInfo := mt.NewInt64ValueRecorder(string(nGetUserInfoValRecorder))
	boundTimeValueRecorders["GetUserInfo"] = mGetUserInfo.Bind(label.String("name", appName))

	return boundTimeValueRecorders
}

func newUserInfoFinderMethodCounters(suffix string, appName string, meter metric.Meter) (boundCounters map[string]metric.BoundInt64Counter) {
	boundCounters = make(map[string]metric.BoundInt64Counter)
	mt := metric.Must(meter)

	nGetUserInfoCounter := []rune("UserInfoFinder_GetUserInfo_" + suffix)
	nGetUserInfoCounter[0] = unicode.ToLower(nGetUserInfoCounter[0])
	cGetUserInfo := mt.NewInt64Counter(string(nGetUserInfoCounter))
	boundCounters["GetUserInfo"] = cGetUserInfo.Bind(label.String("name", appName))

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

		methodTimeMeasure := _d.methodTimeValueRecorders["GetUserInfo"]
		methodTimeMeasure.Record(context.Background(), time.Since(_since).Milliseconds())
	}()
	return _d.base.GetUserInfo(userID)
}
