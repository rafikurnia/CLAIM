package tasks

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rafikurnia/measurement-cli/utils/log"

	"github.com/robfig/cron/v3"
)

var (
	logger        = log.GetLogger("tasks")
	ISOTimeFormat = strings.Replace(time.RFC3339, "Z", "+", 1)
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

func NewSchedule(start, stop, cronExpression string) (*schedule, error) {
	parseTimeFromString := func(inputTime string) *time.Time {
		s := strings.TrimSpace(inputTime)
		if inputTime == "" {
			return &time.Time{}
		}

		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			logger.Errorf("Failed to parse time: %v", err)
			return nil
		}
		return &t
	}

	startTime := parseTimeFromString(start)
	stopTime := parseTimeFromString(stop)
	cronExpr := strings.TrimSpace(cronExpression)

	if startTime == nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse StartTime: '%s'", start))
	}

	if stopTime == nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse StopTime: '%s'", stop))
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(cronExpr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse Cron Expression: '%s'. It is invalid or not supported.", cronExpr))
	}

	schedule := &schedule{StartTime: startTime, StopTime: stopTime, CronExpression: cronExpr}
	err = schedule.validate()
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	return schedule, nil
}
