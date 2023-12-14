package main

import (
	"os"
)

// These variables should be set via LDFLAGS during the build process;
// the justfile in the project global takes care of this automatically.
var (
	Revision  string
	BuildDate string
	Version   string
)

func main() {
	cmd := MainCommand()
	if err := Execute(cmd); err != nil {
		os.Exit(1)
	}
}
