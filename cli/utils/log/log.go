package log

import (
	stdlog "log"
	"os"

	logging "github.com/ipfs/go-log/v2"
)

// List of supported logging levels
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
)

// Static variable to create a log instance for the whole application.
var appLogger = &log{
	level:       LevelDebug,
	initialized: false,
}

// All methods related to logging will belong to this struct.
// This is to reduce the coupling functions and packages.
type log struct {
	level       string
	initialized bool
}

// A non-exported method to initialize a log instance.
func (l *log) init() *log {
	if !l.initialized {
		cfg := &logging.Config{
			Format: logging.ColorizedOutput,
			Level:  logging.LevelError,
			Stderr: false,
			Stdout: true,
		}

		val, present := os.LookupEnv("LOG_LEVEL")
		if present {
			if val != "DEBUG" && val != "INFO" {
				stdlog.Fatalf(
					"Failed to get a logger: %s is an invalid log level.\n"+
						"Supported levels for the application are: [DEBUG INFO]", val,
				)
			}
			l.level = val
		}

		l.initialized = true
		logging.SetupLogging(*cfg)
	}

	return l
}

// An non-exported method to get logger.
func (l *log) getLogger(name string) *logging.ZapEventLogger {
	if !l.initialized {
		stdlog.Panicln("Failed to get a logger: uninitialized")
	}

	logger := logging.Logger(name)
	logging.SetLogLevel(name, l.level)

	return logger
}

// An exported function to initialize and get a logger instance.
func GetLogger(name string) *logging.ZapEventLogger {
	return appLogger.init().getLogger(name)
}
