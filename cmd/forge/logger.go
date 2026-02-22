package main

import (
	"github.com/entrhq/forge/pkg/logging"
)

// cmdLog is the package-level logger for the forge command.
// All startup, configuration, and lifecycle messages should use this logger
// so they are captured in the session log file alongside agent activity.
var cmdLog *logging.Logger

func init() {
	var err error
	cmdLog, err = logging.NewLogger("forge")
	if err != nil {
		// Logger fell back to stderr due to initialization failure.
		cmdLog.Warnf("Failed to initialize forge logger, using stderr fallback: %v", err)
	}
}
