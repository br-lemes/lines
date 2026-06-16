package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/br-lemes/lines/internal/version"
)

func main() {
	version := version.GetVersion()
	// ignore error: may fail on first release when there are no tags yet,
	// any real errors will surface in the next git command
	tag, _ := runOutput("git", "describe", "--tags", "--abbrev=0")
	if strings.TrimSpace(string(tag)) == version {
		return
	}
	err := run("git", "add", "internal/version/version.txt")
	if err != nil {
		panic(err)
	}
	message := fmt.Sprintf("release: %s", version)
	err = run("git", "commit", "-m", message)
	if err != nil {
		panic(err)
	}
	err = run("git", "tag", version)
	if err != nil {
		panic(err)
	}
	err = run("git", "push")
	if err != nil {
		panic(err)
	}
	err = run("git", "push", "--tags")
	if err != nil {
		panic(err)
	}
	err = run(
		"gh", "release", "create", version, "--generate-notes",
		"lines-linux-amd64", "lines-linux-arm64", "lines.exe",
	)
	if err != nil {
		panic(err)
	}
}

func runOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
