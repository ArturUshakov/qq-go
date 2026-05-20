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

var NoColor = os.Getenv("NO_COLOR") != ""

func colorize(color string, value string) string {
	if NoColor {
		return value
	}
	return color + value + reset
}

func Info(format string, args ...any)    { fmt.Printf(colorize(cyan, format)+"\n", args...) }
func Success(format string, args ...any) { fmt.Printf(colorize(green, format)+"\n", args...) }
func Warn(format string, args ...any)    { fmt.Printf(colorize(yellow, format)+"\n", args...) }
func Error(format string, args ...any)   { fmt.Fprintf(os.Stderr, colorize(red, format)+"\n", args...) }
func Title(format string, args ...any)   { fmt.Printf(colorize(bold+blue, format)+"\n", args...) }
func Plain(format string, args ...any)   { fmt.Printf(format+"\n", args...) }
