package main

import (
	_ "embed"
	"os"

	"github.com/br-lemes/lines/cmd"
)

//go:embed .version
var version string

func main() {
	err := cmd.Execute(version)
	if err != nil {
		os.Exit(1)
	}
}
