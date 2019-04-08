package schedule_test

import (
	"github.com/alexandre-normand/slackscot/schedule"
	"github.com/marcsantiago/gocron"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScheduleDefinitionString(t *testing.T) {
	scheduleDefinitionToString := []struct {
		sd             schedule.Definition
		friendlyString string
	}{
		{schedule.Definition{Interval: 1, Weekday: time.Monday.String(), AtTime: "10:00"}, "Every Monday at 10:00"},
		{schedule.Definition{Interval: 1, Weekday: time.Tuesday.String(), AtTime: "09:00"}, "Every Tuesday at 09:00"},
		{schedule.Definition{Interval: 1, Weekday: time.Wednesday.String(), AtTime: "08:00"}, "Every Wednesday at 08:00"},
		{schedule.Definition{Interval: 1, Weekday: time.Thursday.String(), AtTime: "07:00"}, "Every Thursday at 07:00"},
		{schedule.Definition{Interval: 1, Weekday: time.Friday.String(), AtTime: "06:00"}, "Every Friday at 06:00"},
		{schedule.Definition{Interval: 1, Weekday: time.Saturday.String(), AtTime: "05:00"}, "Every Saturday at 05:00"},
		{schedule.Definition{Interval: 1, Weekday: time.Sunday.String(), AtTime: "04:00"}, "Every Sunday at 04:00"},
		{schedule.Definition{Interval: 1, Unit: schedule.Seconds}, "Every second"},
		{schedule.Definition{Interval: 2, Unit: schedule.Seconds}, "Every 2 seconds"},
		{schedule.Definition{Interval: 1, Unit: schedule.Minutes}, "Every minute"},
		{schedule.Definition{Interval: 2, Unit: schedule.Minutes}, "Every 2 minutes"},
		{schedule.Definition{Interval: 1, Unit: schedule.Hours}, "Every hour"},
		{schedule.Definition{Interval: 2, Unit: schedule.Hours}, "Every 2 hours"},
		{schedule.Definition{Interval: 1, Unit: schedule.Days}, "Every day"},
		{schedule.Definition{Interval: 2, Unit: schedule.Days}, "Every 2 days"},
		{schedule.Definition{Interval: 1, Unit: schedule.Days, AtTime: "10:00"}, "Every day at 10:00"},
		{schedule.Definition{Interval: 2, Unit: schedule.Days, AtTime: "10:00"}, "Every 2 days at 10:00"},
		{schedule.Definition{Interval: 1, Unit: schedule.Weeks}, "Every week"},
		{schedule.Definition{Interval: 2, Unit: schedule.Weeks}, "Every 2 weeks"},
	}

	for _, testCase := range scheduleDefinitionToString {
		t.Run(testCase.friendlyString, func(t *testing.T) {
			friendlyStr := testCase.sd.String()
			assert.Equalf(t, testCase.friendlyString, friendlyStr, "Expected different string value for schedule definition: %v", testCase.sd)
		})
	}
}

func TestScheduleDefinitionBuilder(t *testing.T) {
	scheduleDefinitionToString := []struct {
		sd             schedule.Definition
		friendlyString string
	}{
		{schedule.New().Every(time.Monday.String()).AtTime("10:00").Build(), "Every Monday at 10:00"},
		{schedule.New().WithUnit(schedule.Seconds).Build(), "Every second"},
		{schedule.New().WithInterval(2, schedule.Seconds).Build(), "Every 2 seconds"},
		{schedule.New().Every(time.Monday.String()).Build(), "Every Monday"},
	}

	for _, testCase := range scheduleDefinitionToString {
		t.Run(testCase.friendlyString, func(t *testing.T) {
			friendlyStr := testCase.sd.String()
			assert.Equalf(t, testCase.friendlyString, friendlyStr, "Expected different string value for schedule definition: %v", testCase.sd)
		})
	}
}

func TestNewScheduledJobFromScheduleDefinition(t *testing.T) {
	scheduleDefinitionToResult := []struct {
		sd           schedule.Definition
		errorMessage string
	}{
		{schedule.Definition{Interval: 1, Weekday: time.Monday.String(), AtTime: "10:00"}, ""},
		{schedule.Definition{Interval: 1, Weekday: time.Tuesday.String(), AtTime: "09:00"}, ""},
		{schedule.Definition{Interval: 1, Weekday: time.Wednesday.String(), AtTime: "08:00"}, ""},
		{schedule.Definition{Interval: 1, Weekday: time.Thursday.String(), AtTime: "07:00"}, ""},
		{schedule.Definition{Interval: 1, Weekday: time.Friday.String(), AtTime: "06:00"}, ""},
		{schedule.Definition{Interval: 1, Weekday: time.Saturday.String(), AtTime: "05:00"}, ""},
		{schedule.Definition{Interval: 1, Weekday: time.Sunday.String(), AtTime: "04:00"}, ""},
		{schedule.Definition{Interval: 1, Unit: schedule.Seconds}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Seconds}, ""},
		{schedule.Definition{Interval: 1, Unit: schedule.Minutes}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Minutes}, ""},
		{schedule.Definition{Interval: 1, Unit: schedule.Hours}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Hours}, ""},
		{schedule.Definition{Interval: 1, Unit: schedule.Days}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Days}, ""},
		{schedule.Definition{Interval: 1, Unit: schedule.Days, AtTime: "10:00"}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Days, AtTime: "10:00"}, ""},
		{schedule.Definition{Interval: 1, Unit: schedule.Weeks}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Weeks}, ""},
		{schedule.Definition{Interval: 2, Unit: schedule.Weeks, Weekday: time.Monday.String()}, ""}, // When we have a weekday, we ignore units so it's still valid
		{schedule.Definition{Interval: 1, Unit: schedule.Seconds, AtTime: "10:00"}, "Can't run job on schedule [Every second at 10:00] with AtTime in conjunction with a sub-day IntervalUnit"},
		{schedule.Definition{Interval: 1, Unit: schedule.Minutes, AtTime: "10:00"}, "Can't run job on schedule [Every minute at 10:00] with AtTime in conjunction with a sub-day IntervalUnit"},
		{schedule.Definition{Interval: 1, Unit: schedule.Hours, AtTime: "10:00"}, "Can't run job on schedule [Every hour at 10:00] with AtTime in conjunction with a sub-day IntervalUnit"},
	}

	scheduler := gocron.NewScheduler()
	for _, testCase := range scheduleDefinitionToResult {
		t.Run(testCase.sd.String(), func(t *testing.T) {

			_, err := schedule.NewJob(scheduler, testCase.sd)

			if testCase.errorMessage == "" {
				assert.Nilf(t, err, "Expected valid job to be created for schedule definition: %v", testCase.sd)
			} else {
				if assert.Error(t, err) {
					assert.Equal(t, testCase.errorMessage, err.Error())
				}
			}
		})
	}
}
