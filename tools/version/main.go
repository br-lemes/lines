package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func version() string {
	major := 0
	minor := 0
	patch := 0

	cmd := exec.Command("git", "log", "--pretty=format:%s")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	}

	logs := strings.Split(string(output), "\n")
	breakingChangeRegex := regexp.MustCompile(`^[a-z]+!:\s*`)

	for i := len(logs) - 1; i >= 0; i-- {
		line := strings.TrimSpace(logs[i])
		if line == "" {
			continue
		}

		if breakingChangeRegex.MatchString(line) {
			major++
			minor = 0
			patch = 0
			continue
		}

		if strings.HasPrefix(line, "feat") {
			minor++
			patch = 0
			continue
		}

		if strings.HasPrefix(line, "fix") {
			patch++
			continue
		}
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

func main() {
	version := version()
	err := os.WriteFile("internal/version/version.txt", []byte(version), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing version file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(version)
}
