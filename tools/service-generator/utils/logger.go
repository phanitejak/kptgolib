package utils

import "fmt"

type logLevel int

// Log levels for logger
const (
	INFO logLevel = iota
	SUCCESS
	ERROR
)

// Log prints the message with specified level to standard output
func Log(message string, level logLevel) {
	switch level {
	case INFO:
		message = fmt.Sprintf("%v[ℹ]  %v %v\n", ColorBlue, message, ColorReset)
	case SUCCESS:
		message = fmt.Sprintf("%v[✔]  %v %v\n", ColorGreen, message, ColorReset)
	case ERROR:
		message = fmt.Sprintf("%v[✖]  %v %v\n", ColorRed, message, ColorReset)
	}
	fmt.Print(message)
}
