package main

import (
	"os"

	"github.com/br-lemes/lines/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
