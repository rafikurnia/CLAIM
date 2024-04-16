package tasks

import (
	"fmt"
	"os"
	"time"
)

type Task struct {
	MeasurementStartTime time.Time
	MeasurementStopTime  time.Time
	Region               string
	Result               string
	Sequence             int
}

func NewTask() (*Task, error) {
	// n, err := GetNodeInfo()
	// if err != nil {
	// 	err = fmt.Errorf("GetNodeInfo -> %w", err)
	// }

	// if n == nil {
	// 	n = &nodeInfo{}
	// }

	return &Task{
		// MeasurerInfo:         n,
		MeasurementStartTime: time.Time{},
		MeasurementStopTime:  time.Time{},
		Region:               os.Getenv("REGION"),
	}, nil
}

type TaskMetadata struct {
	VantagePoints    []string
	Probe            string
	Arguments        string
	Schedule         *schedule
	Type             string
	Status           string
	NumberOfSequence map[string]int
}

func NewTaskMetadata() (*TaskMetadata, error) {
	s, err := NewSchedule("", "", "* * * * *")
	if err != nil {
		return nil, fmt.Errorf("NewSchedule -> %w", err)
	}

	return &TaskMetadata{
		VantagePoints:    make([]string, 0),
		Schedule:         s,
		NumberOfSequence: make(map[string]int),
	}, nil
}
