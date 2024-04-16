package p

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type schedule struct {
	StartTime      *time.Time
	StopTime       *time.Time
	CronExpression string
}

// Validate user input about the start and stop time.
func (s *schedule) validate() error {
	if !s.StartTime.IsZero() &&
		s.StartTime.Before(time.Now()) {
		return errors.New("the StartTime is in the past")
	}

	if !s.StopTime.IsZero() &&
		s.StopTime.Before(time.Now()) {
		return errors.New("the StopTime is in the past")
	}

	if !s.StartTime.IsZero() &&
		!s.StopTime.IsZero() &&
		s.StopTime.Before(*s.StartTime) {
		return errors.New("the StopTime is before the StartTime")
	}

	return nil
}

func parseTimeFromString(inputTime string) (*time.Time, error) {
	s := strings.TrimSpace(inputTime)
	if inputTime == "" {
		return &time.Time{}, nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, fmt.Errorf("time.Parse -> %w", err)
	}
	return &t, nil
}

func newSchedule(start, stop, cronExpression string) (*schedule, error) {
	startTime, err := parseTimeFromString(start)
	if err != nil {
		return nil, fmt.Errorf("failed to parse StartTime -> '%w'", err)
	}

	stopTime, err := parseTimeFromString(stop)
	if err != nil {
		return nil, fmt.Errorf("failed to parse StopTime -> '%w'", err)
	}

	cronExpr := strings.TrimSpace(cronExpression)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err = parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Cron Expression: '%s'. It is invalid or not supported -> %w", cronExpr, err)
	}

	schedule := &schedule{StartTime: startTime, StopTime: stopTime, CronExpression: cronExpr}
	err = schedule.validate()
	if err != nil {
		return nil, fmt.Errorf("validate -> %w", err)
	}
	return schedule, nil
}
