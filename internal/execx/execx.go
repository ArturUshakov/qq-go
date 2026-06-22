package execx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Result struct {
	Stdout string
	Stderr string
	Code   int
}

func Exists(name string) bool { _, err := exec.LookPath(name); return err == nil }

func Output(name string, args ...string) (Result, error) {
	return outputWithInput(nil, name, args...)
}

func OutputWithInput(input []byte, name string, args ...string) (Result, error) {
	return outputWithInput(input, name, args...)
}

func outputWithInput(input []byte, name string, args ...string) (Result, error) {
	cmd := exec.Command(name, args...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), Code: 0}
	if err == nil {
		return result, nil
	}
	result.Code = -1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.Code = exitErr.ExitCode()
	}
	message := strings.TrimSpace(result.Stderr)
	if message == "" {
		return result, fmt.Errorf("не удалось выполнить %s: %w", name, err)
	}
	return result, fmt.Errorf("не удалось выполнить %s: %w: %s", name, err, message)
}

func RunInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}

func RunWithTerminal(terminal *os.File, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = terminal
	cmd.Stdout = terminal
	cmd.Stderr = terminal
	return cmd.Run()
}
func RunQuiet(name string, args ...string) error { return exec.Command(name, args...).Run() }
func RunPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
