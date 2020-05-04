package gocommandutil

import (
	"bufio"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// Now I have implemented command stuff so many times that I will put them
// here successively when I need them

// ExecuteCmd execute command with time out
// stdoutHandler and stderrHandler is called on each line of output
func ExecuteCmd(timeoutsecs int,
	stdoutHandler func(string) error,
	stderrHandler func(string) error,
	command string,

	args ...string) error {

	cmd := exec.Command(command, args...)

	// Force the child processes to start in it's own process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("StdoutPipe(): %v", err)
	}

	defer stdOut.Close()

	scanner := bufio.NewScanner(stdOut)
	go func() {
		for scanner.Scan() {
			stdoutHandler(scanner.Text())
		}
	}()

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("StderrPipe(): %v", err)
	}

	defer stdErr.Close()

	stdErrScanner := bufio.NewScanner(stdErr)
	go func() {
		for stdErrScanner.Scan() {
			stdoutHandler(stdErrScanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting Cmd: %v", err)
	}

	// Use a channel to signal completion so we can use a select statement
	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	// Start a timer
	timeout := time.After(time.Duration(timeoutsecs) * time.Second)

	// The select statement allows us to execute based on which channel
	// we get a message from first.
	select {
	case <-timeout:
		// Timeout happened first, kill the process and print a message.
		cmd.Process.Kill()
		return fmt.Errorf("Timed out on Command: %v", cmd)
	case err := <-done:
		// Command completed before timeout. Print output and error if it exists.
		if err != nil {
			return fmt.Errorf("executeCmd: %s: %v", command, err)
		}
	}
	return nil
}
