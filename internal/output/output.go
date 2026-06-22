package output

import (
	"fmt"
	"os"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
)

func IsTerminal(file *os.File) bool {
	if os.Getenv("CI") != "" || os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func colorize(file *os.File, color string, value string) string {
	if !IsTerminal(file) {
		return value
	}
	return color + value + reset
}

func Info(format string, args ...any) {
	fmt.Fprintf(os.Stdout, colorize(os.Stdout, cyan, format)+"\n", args...)
}
func Success(format string, args ...any) {
	fmt.Fprintf(os.Stdout, colorize(os.Stdout, green, format)+"\n", args...)
}
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(os.Stderr, yellow, format)+"\n", args...)
}
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(os.Stderr, red, format)+"\n", args...)
}
func Title(format string, args ...any) {
	fmt.Fprintf(os.Stdout, colorize(os.Stdout, bold+blue, format)+"\n", args...)
}
func Plain(format string, args ...any) { fmt.Fprintf(os.Stdout, format+"\n", args...) }
