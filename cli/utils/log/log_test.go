package log

import (
	"testing"

	logging "github.com/ipfs/go-log/v2"

	"github.com/stretchr/testify/assert"
)

func TestLogConfigAfterInitialized(t *testing.T) {
	GetLogger("testlog") // Call to initialize logger

	cfg := logging.GetConfig()

	expectedFormat := logging.ColorizedOutput
	expectedLevel := logging.LevelError
	expectedStderr := false
	expectedStdout := true

	assert.Equal(t, expectedFormat, cfg.Format, "The log format must be the same.")
	assert.Equal(t, expectedLevel, cfg.Level, "The log level must be the same.")
	assert.Equal(t, expectedStderr, cfg.Stderr, "The log stderr must be the same.")
	assert.Equal(t, expectedStdout, cfg.Stdout, "The log stdout must be the same.")
}

func TestPanicWhenUninitialized(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	// create a new instance of logger
	var testLogger = &log{
		level:       LevelDebug,
		initialized: false,
	}

	// Should throw panic because uninitialized
	testLogger.getLogger("Test")
}
