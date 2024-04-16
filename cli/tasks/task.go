package tasks

type Task struct {
	VantagePoints []string
	Probe         string
	Arguments     string
	Schedule      *schedule
}

func NewTask() *Task {
	s, _ := NewSchedule("", "", "* * * * *")

	return &Task{
		VantagePoints: make([]string, 0),
		Schedule:      s,
	}
}

type TaskManagement struct {
	Action string
	TaskID string
}
