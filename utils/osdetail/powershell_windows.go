package osdetail

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// RunPowershellCmd runs a powershell command and returns its output.
func RunPowershellCmd(script string) (output string, err error) {
	// Create command to execute.
	cmd := exec.Command(
		"powershell.exe",
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-NonInteractive",
		"[System.Console]::OutputEncoding = [System.Text.Encoding]::UTF8\n"+script,
	)

	// Create and assign output buffers.
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Run command and collect output.
	err = cmd.Run()
	stdout, stderr := stdoutBuf.String(), stderrBuf.String()
	if err != nil {
		return "", err
	}
	// Powershell might not return an error, but just write to stdout instead.
	if stderr != "" {
		return "", errors.New(strings.SplitN(stderr, "\n", 2)[0])
	}

	// Debugging output:
	// fmt.Printf("powershell stdout: %s\n", stdout)
	// fmt.Printf("powershell stderr: %s\n", stderr)

	// Finalize stdout.
	cleanedOutput := strings.TrimSpace(stdout)
	if cleanedOutput == "" {
		return "", ErrEmptyOutput
	}

	return cleanedOutput, nil
}
