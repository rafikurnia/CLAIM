package tasks

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
		return errors.New("The StartTime is in the past.")
	}

	if !s.StopTime.IsZero() &&
		s.StopTime.Before(time.Now()) {
		return errors.New("The StopTime is in the past.")
	}

	if !s.StartTime.IsZero() &&
		!s.StopTime.IsZero() &&
		s.StopTime.Before(*s.StartTime) {
		return errors.New("The StopTime is before the StartTime.")
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

func NewSchedule(start, stop, cronExpression string) (*schedule, error) {
	startTime, err := parseTimeFromString(start)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse StartTime: '%s'", start)
	}

	stopTime, err := parseTimeFromString(stop)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse StopTime: '%s'", stop)
	}

	cronExpr := strings.TrimSpace(cronExpression)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err = parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse Cron Expression: '%s'. It is invalid or not supported.", cronExpr)
	}

	schedule := &schedule{StartTime: startTime, StopTime: stopTime, CronExpression: cronExpr}

	if err := schedule.validate(); err != nil {
		return nil, fmt.Errorf("schedule.validate -> %w", err)
	}
	return schedule, nil
}
