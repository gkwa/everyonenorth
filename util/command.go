package util

import (
	"bytes"
	"os/exec"
)

func RunCommand(cmd *exec.Cmd, cwd string) (output string, exitCode int, err error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if cwd != "" {
		cmd.Dir = cwd
	}

	err = cmd.Run()
	exitCode = cmd.ProcessState.ExitCode()
	output = stdout.String()

	if err != nil {
		output += "\n" + stderr.String()
	}

	return output, exitCode, err
}
