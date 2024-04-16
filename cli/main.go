package main

import (
	"github.com/rafikurnia/measurement-cli/connections"
	"github.com/rafikurnia/measurement-cli/tasks"
	"github.com/rafikurnia/measurement-cli/utils/flag"
	"github.com/rafikurnia/measurement-cli/utils/log"
)

func main() {
	// Get a logger for main package.
	logger := log.GetLogger("main")

	// Get configurations from program flags.
	subcommand, cfg := flag.Parse()
	logger.Debug("Parsed program flags")

	switch subcommand {
	case flag.MeasureCommand:
		connections.SendTask(cfg.(*tasks.Task))

	case flag.ManageCommand:
		task := cfg.(*tasks.TaskManagement)

		switch task.Action {
		case "status":
			logger.Debugf("Checking the status of task with ID: %s", task.TaskID)
			connections.GetStatus(task.TaskID)

		case "results":
			logger.Debugf("Get the results of a task with ID: %s", task.TaskID)
			connections.GetResults(task.TaskID)

		case "cancel":
			logger.Debugf("Cancel the task with ID: %s", task.TaskID)
			connections.CancelTask(task.TaskID)
		}
	}
}
