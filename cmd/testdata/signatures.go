package main

import "fmt"

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

	callback := func(status int, message string) {
		fmt.Println(message)
	}

	callback(200, "success")
}
