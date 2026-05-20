package main

import (
	"os"

	"github.com/michaelin/agent-memory/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
