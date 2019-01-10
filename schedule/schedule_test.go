package schedule_test

import (
	"github.com/alexandre-normand/slackscot/v2/schedule"
	"github.com/marcsantiago/gocron"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScheduleDefinitionString(t *testing.T) {
	scheduleDefinitionToString := []struct {
		sd             schedule.ScheduleDefinition
		friendlyString string
	}{
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Monday.String(), AtTime: "10:00"}, "Every Monday at 10:00"},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Tuesday.String(), AtTime: "09:00"}, "Every Tuesday at 09:00"},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Wednesday.String(), AtTime: "08:00"}, "Every Wednesday at 08:00"},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Thursday.String(), AtTime: "07:00"}, "Every Thursday at 07:00"},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Friday.String(), AtTime: "06:00"}, "Every Friday at 06:00"},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Saturday.String(), AtTime: "05:00"}, "Every Saturday at 05:00"},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Sunday.String(), AtTime: "04:00"}, "Every Sunday at 04:00"},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Seconds}, "Every second"},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Seconds}, "Every 2 seconds"},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Minutes}, "Every minute"},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Minutes}, "Every 2 minutes"},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Hours}, "Every hour"},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Hours}, "Every 2 hours"},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Days}, "Every day"},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Days}, "Every 2 days"},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Days, AtTime: "10:00"}, "Every day at 10:00"},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Days, AtTime: "10:00"}, "Every 2 days at 10:00"},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Weeks}, "Every week"},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Weeks}, "Every 2 weeks"},
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
		sd           schedule.ScheduleDefinition
		valid        bool
		errorMessage string
	}{
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Monday.String(), AtTime: "10:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Tuesday.String(), AtTime: "09:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Wednesday.String(), AtTime: "08:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Thursday.String(), AtTime: "07:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Friday.String(), AtTime: "06:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Saturday.String(), AtTime: "05:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Weekday: time.Sunday.String(), AtTime: "04:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Seconds}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Seconds}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Minutes}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Minutes}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Hours}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Hours}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Days}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Days}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Days, AtTime: "10:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Days, AtTime: "10:00"}, true, ""},
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Weeks}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Weeks}, true, ""},
		{schedule.ScheduleDefinition{Interval: 2, Unit: schedule.Weeks, Weekday: time.Monday.String()}, true, ""}, // When we have a weekday, we ignore units so it's still valid
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Seconds, AtTime: "10:00"}, true, ""},             // gocron just ignores AtTime when not relevant to the unit
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Minutes, AtTime: "10:00"}, true, ""},             // gocron just ignores AtTime when not relevant to the unit
		{schedule.ScheduleDefinition{Interval: 1, Unit: schedule.Hours, AtTime: "10:00"}, true, ""},               // gocron just ignores AtTime when not relevant to the unit
	}

	scheduler := gocron.NewScheduler()
	for _, testCase := range scheduleDefinitionToResult {
		t.Run(testCase.sd.String(), func(t *testing.T) {

			_, err := schedule.NewJob(scheduler, testCase.sd)

			if testCase.valid {
				assert.Nilf(t, err, "Expected valid job to be created for schedule definition: %v", testCase.sd)
			} else {
				if assert.NotNil(t, err) {
					assert.Contains(t, err.Error(), testCase.errorMessage)
				}
			}
		})
	}
}
