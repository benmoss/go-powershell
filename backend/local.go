// Copyright (c) 2017 Gorillalabs. All rights reserved.

package backend

import (
	"fmt"
	"io"
	"os/exec"
)

type Local struct{}

func (b *Local) StartProcess(cmd string, args ...string) (Waiter, io.Writer, io.Reader, io.Reader, error) {
	command := exec.Command(cmd, args...)

	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not get hold of the PowerShell's stdin stream: %v", err)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not get hold of the PowerShell's stdout stream: %v", err)
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not get hold of the PowerShell's stderr stream: %v", err)
	}

	err = command.Start()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("could not spawn PowerShell process: %v", err)
	}

	return command, stdin, stdout, stderr, nil
}
