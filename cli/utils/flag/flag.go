package flag

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/rafikurnia/measurement-cli/tasks"
	"github.com/rafikurnia/measurement-cli/utils/log"
)

const (
	MeasureCommand = "measure"
	ManageCommand  = "manage"
)

// Parse configuration based on the user input.
func Parse() (string, interface{}) {
	printUsage := func() {
		fmt.Printf("Usage: %s <subcommand> [args]\n\n", os.Args[0])

		fmt.Printf("The available subcommands for execution are listed below:\n")
		fmt.Printf("  %s\tSend a measurement task\n", MeasureCommand)
		fmt.Printf("  %s\tManage a measurement task\n", ManageCommand)

		fmt.Printf("\nTo get more information, access help menu on each subcommand as follows:\n")
		fmt.Printf("%s <subcommand> -h\n", os.Args[0])
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		printUsage()
	}

	logger := log.GetLogger("flag")

	switch os.Args[1] {
	case MeasureCommand:
		cfg := tasks.NewTask()

		var startTime, stopTime, cronExpr, vantagePoints string
		subMeasure := flag.NewFlagSet(MeasureCommand, flag.ExitOnError)
		subMeasure.StringVar(&vantagePoints, "r", "", "[required] a comma delimited list of Google Cloud regions")
		subMeasure.StringVar(&cfg.Probe, "p", "", "[required] the ID of the measurement probe (e.g., ping)")
		subMeasure.StringVar(&cfg.Arguments, "a", "", "[required] the arguments for the measurement (e.g., google.com)")
		subMeasure.StringVar(&startTime, "s", "", fmt.Sprintf("the start time of a measurement, in ISO format (e.g., %s) or leave empty for as soon as possible", tasks.ISOTimeFormat))
		subMeasure.StringVar(&stopTime, "e", "", fmt.Sprintf("the end (stop) time of the measurement, in ISO format (e.g., %s) or leave empty for one-off measurement", tasks.ISOTimeFormat))
		subMeasure.StringVar(&cronExpr, "c", "* * * * *", "cron expression (in UTC) to execute the measurement, it is ignored on one-off measurement")

		subMeasure.Parse(os.Args[2:])
		if subMeasure.Parsed() {
			vantagePoints = strings.TrimSpace(vantagePoints)
			cfg.Probe = strings.TrimSpace(cfg.Probe)
			cfg.Arguments = strings.TrimSpace(cfg.Arguments)

			isError := false
			if vantagePoints == "" {
				logger.Error("The list of vantage points (i.e., a list of Google Cloud region names) cannot be empty.")
				isError = true
			}

			for _, v := range strings.Split(vantagePoints, ",") {
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					if strings.ToLower(trimmed) == "all" {
						cfg.VantagePoints = maps.Keys(regions)
						break
					}

					if _, ok := regions[trimmed]; !ok {
						logger.Errorf("The region is invalid and will be ignored: '%s'.", trimmed)
					} else {
						cfg.VantagePoints = append(cfg.VantagePoints, trimmed)
					}
				}
			}

			if len(cfg.VantagePoints) == 0 {
				logger.Errorf("No valid vantage point is specified. Valid values are : [%v].", strings.Join(maps.Keys(regions), "|"))
				isError = true
			}

			if cfg.Probe == "" {
				logger.Error("The measurement probe cannot be empty.")
				isError = true
			}

			if _, ok := probes[cfg.Probe]; !ok {
				logger.Errorf("The specified measurement probe is not supported. Supported values are: [%v].", strings.Join(maps.Keys(probes), "|"))
				isError = true
			}

			if cfg.Arguments == "" {
				logger.Error("The arguments for the measurement probe cannot be empty.")
				isError = true
			}

			if cfg.Probe == "httpstat" &&
				!strings.HasPrefix(cfg.Arguments, "http://") &&
				!strings.HasPrefix(cfg.Arguments, "https://") {
				logger.Error("The arguments must contain URL starts with either 'http://' or 'https://'.")
				isError = true
			}

			schedule, err := tasks.NewSchedule(startTime, stopTime, cronExpr)
			if err != nil {
				logger.Errorf("An error occurred when parsing schedule: %v", err)
				isError = true
			}

			if isError {
				fmt.Printf("Usage: %s measure [args]\n\n", os.Args[0])
				subMeasure.PrintDefaults()
				os.Exit(1)
			}

			cfg.Schedule = schedule
			logger.Debugf("The flags have been parsed: %+v", cfg)
		} else {
			logger.Fatal("An error occurred when parsing program flags")
		}

		return os.Args[1], cfg

	case ManageCommand:
		cfg := &tasks.TaskManagement{}

		subManage := flag.NewFlagSet(ManageCommand, flag.ExitOnError)
		subManage.StringVar(&cfg.Action, "a", "", "[required] The management action for a task values=[cancel|results|status].\n"+
			"\t   - cancel\tcancel measurement task\n"+
			"\t   - results\tget measurement results\n"+
			"\t   - status\tget measurement status",
		)

		subManage.StringVar(&cfg.TaskID, "t", "", "[required] The ID of the measurement task to manage.")

		subManage.Parse(os.Args[2:])
		if subManage.Parsed() {
			cfg.Action = strings.TrimSpace(cfg.Action)
			cfg.TaskID = strings.TrimSpace(cfg.TaskID)

			isError := false
			if cfg.Action == "" {
				logger.Error("The action cannot be empty.")
				isError = true
			}

			if cfg.Action != "results" && cfg.Action != "status" && cfg.Action != "cancel" {
				logger.Error("The specified action is not supported. The supported values are: [cancel|results|status].")
				isError = true
			}

			if cfg.TaskID == "" {
				logger.Error("The taskID cannot be empty.")
				isError = true
			}

			if isError {
				fmt.Printf("Usage: %s manage [args]\n\n", os.Args[0])
				subManage.PrintDefaults()
				os.Exit(1)
			}
		} else {
			logger.Fatal("An error occurred when parsing program flags")
		}

		return os.Args[1], cfg
	default:
		printUsage()
	}

	return "", nil
}
