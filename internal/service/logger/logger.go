package logger

import (
	"fmt"
	"log"
)

// Define color constants for better readability
const (
	ColorReset  = "\033[0m"  // Reset the color to default
	ColorBlue   = "\033[34m" // Blue
	ColorGreen  = "\033[32m" // Green
	ColorYellow = "\033[33m" // Yellow
	ColorRed    = "\033[31m" // Red
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (*Logger) Print(level string, message string) {
	log.Print(buildMessage(level, message))
}

func (*Logger) Fatal(level string, message string) {
	log.Fatal(buildMessage(level, message))
}

func buildMessage(level string, message string) string {
	var color string

	// Set color based on log level using a switch-case
	switch level {
	case "DEBUG":
		color = ColorBlue
	case "INFO":
		color = ColorGreen
	case "WARN":
		color = ColorYellow
	case "ERROR":
		color = ColorRed
	default:
		color = ColorReset
	}

	return fmt.Sprintf("%s%s: %s%s", color, level, message, ColorReset)
}
