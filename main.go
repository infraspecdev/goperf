package main

import (
	"os"

	"github.com/infraspecdev/goperf/cmd"
)

var version = "dev"

func main() {
	if err := cmd.NewRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}
