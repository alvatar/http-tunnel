package verbose

import (
	"fmt"
	"time"
)

var verbose bool = false

func SetVerbose() {
	verbose = true
}

// Get timestamp for logging purposes
func ts() string {
	return time.Now().Format("2006-01-02 15:04:05.999 -0700 MST")
}

// Time-stamped printf
func TSPrintf(format string, a ...interface{}) {
	fmt.Printf("%s ", ts())
	fmt.Printf(format, a...)
}

// print debug/status conditionally on having Verbose on
func Log(format string, a ...interface{}) {
	if verbose {
		TSPrintf(format, a...)
	}
}
