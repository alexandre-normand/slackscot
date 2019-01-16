// Package schedule defines the interface for scheduling of slackscot actions
package schedule

import (
	"fmt"
	"github.com/marcsantiago/gocron"
	"strings"
	"time"
)

// Definition holds the data defining a schedule definition
type Definition struct {
	// Internal value (every 1 minute would be expressed with an interval of 1). Must be set explicitly or implicitly (a weekday value implicitly sets the interval to 1)
	Interval uint64

	// Must be set explicitly or implicitly ("weeks" is implicitly set when "Weekday" is set). Valid time units are: "weeks", "hours", "days", "minutes", "seconds"
	Unit string

	// Optional day of the week. If set, unit and interval are ignored and implicitly considered to be "every 1 week"
	Weekday string

	// Optional "at time" value (i.e. "10:30")
	AtTime string
}

// Unit values
const (
	Weeks   = "weeks"
	Hours   = "hours"
	Days    = "days"
	Minutes = "minutes"
	Seconds = "seconds"
)

var weekdayToNumeral = map[string]time.Weekday{
	time.Monday.String():    time.Monday,
	time.Tuesday.String():   time.Tuesday,
	time.Wednesday.String(): time.Wednesday,
	time.Thursday.String():  time.Thursday,
	time.Friday.String():    time.Friday,
	time.Saturday.String():  time.Saturday,
	time.Sunday.String():    time.Sunday,
}

// Returns a human-friendly string for the schedule definition
func (d Definition) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Every ")

	if d.Weekday != "" {
		fmt.Fprintf(&b, "%s", d.Weekday)
	} else if d.Interval == 1 {
		fmt.Fprintf(&b, "%s", strings.TrimSuffix(d.Unit, "s"))
	} else {
		fmt.Fprintf(&b, "%d %s", d.Interval, d.Unit)
	}

	if d.AtTime != "" {
		fmt.Fprintf(&b, " at %s", d.AtTime)
	}

	return b.String()
}

// Option defines an option for a Slackscot
type scheduleOption func(j *gocron.Job)

// optionWeekday sets the weekday of a recurring job
func optionWeekday(weekday string) func(j *gocron.Job) {
	return func(j *gocron.Job) {
		switch weekday {
		case time.Monday.String():
			j = j.Monday()
		case time.Tuesday.String():
			j = j.Tuesday()
		case time.Wednesday.String():
			j = j.Wednesday()
		case time.Thursday.String():
			j = j.Thursday()
		case time.Friday.String():
			j = j.Friday()
		case time.Saturday.String():
			j = j.Saturday()
		case time.Sunday.String():
			j = j.Sunday()
		}
	}
}

// optionUnit sets the unit of a recurring job
func optionUnit(unit string) func(j *gocron.Job) {
	return func(j *gocron.Job) {
		switch unit {
		case Weeks:
			j = j.Weeks()
		case Hours:
			j = j.Hours()
		case Days:
			j = j.Days()
		case Minutes:
			j = j.Minutes()
		case Seconds:
			j = j.Seconds()
		}
	}
}

// optionAtTime sets the AtTime of a recurring job
func optionAtTime(atTime string) func(j *gocron.Job) {
	return func(j *gocron.Job) {
		j = j.At(atTime)
	}
}

// NewJob sets up the gocron.Job with the schedule and leaves the task undefined for the caller to set up
func NewJob(s *gocron.Scheduler, def Definition) (j *gocron.Job, err error) {
	j = s.Every(def.Interval, false)

	scheduleOptions := make([]scheduleOption, 0)

	if def.Weekday != "" {
		scheduleOptions = append(scheduleOptions, optionWeekday(def.Weekday))
	} else if def.Unit != "" {
		scheduleOptions = append(scheduleOptions, optionUnit(def.Unit))
	}

	if def.AtTime != "" {
		scheduleOptions = append(scheduleOptions, optionAtTime(def.AtTime))
	}

	for _, option := range scheduleOptions {
		option(j)
	}

	if j.Err() != nil {
		return nil, j.Err()
	}

	return j, nil
}
