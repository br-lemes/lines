package main

import "fmt"

type CommandRunner func(commandName string, options []string, timeoutSeconds int) (string, error)

type WorkerConfig struct {
	Handler func(payload string, retryCount int) error
}

type EventProcessor interface {
	ProcessEvent(id int, data []byte, force bool) bool
}

func ExecuteSignatureCheck() {
	short := 1
	long := 2
	fmt.Printf("short: %d\nlong: %d\n", short, long)
}

func main() {
	ExecuteSignatureCheck()

	callback := func(status int, message string) { fmt.Println(message) }

	callback(200, "success")

	var runner CommandRunner
	runner = func(commandName string, options []string, timeoutSeconds int) (string, error) {
		return commandName, nil
	}

	result, err := runner("lines", []string{"--check-signatures"}, 30)
	if err == nil {
		fmt.Println(result)
	}
}
