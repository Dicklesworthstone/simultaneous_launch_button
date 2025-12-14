// Package main provides the entry point for the SLB (Simultaneous Launch Button) CLI.
// SLB implements a two-person rule system for dangerous command authorization.
package main

import (
	"os"

	"github.com/Dicklesworthstone/slb/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
