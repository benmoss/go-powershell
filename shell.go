// Copyright (c) 2017 Gorillalabs. All rights reserved.

package powershell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/bhendo/go-powershell/backend"
	"github.com/bhendo/go-powershell/utils"
	"github.com/juju/errors"
)

const newline = "\r\n"

type Shell interface {
	Execute(cmd string) (string, string, error)
	Exit()
}

type shell struct {
	handle   backend.Waiter
	stdin    io.Writer
	stdout   io.Reader
	stderr   io.Reader
	debugerr []byte
	debugout []byte
}

func New(backend backend.Starter) (Shell, error) {
	handle, stdin, stdout, stderr, err := backend.StartProcess("powershell.exe", "-NoExit", "-Command", "-")
	if err != nil {
		return nil, err
	}

	return &shell{handle, stdin, stdout, stderr, []byte{}, []byte{}}, nil
}

func (s *shell) Execute(cmd string) (string, string, error) {
	if s.handle == nil {
		return "", "", errors.Annotate(errors.New(cmd), "Cannot execute commands on closed shells.")
	}

	outBoundary := createBoundary()
	errBoundary := createBoundary()

	// wrap the command in special markers so we know when to stop reading from the pipes
	full := fmt.Sprintf("%s; echo '%s'; [Console]::Error.WriteLine('%s')%s", cmd, outBoundary, errBoundary, newline)

	_, err := s.stdin.Write([]byte(full))
	if err != nil {
		return "", "", errors.Annotate(errors.Annotate(err, cmd), "Could not send PowerShell command")
	}

	// read stdout and stderr
	sout := ""
	serr := ""

	waiter := &sync.WaitGroup{}
	waiter.Add(2)

	go s.streamReader(s.stdout, outBoundary, &sout, waiter, &s.debugout)
	go s.streamReader(s.stderr, errBoundary, &serr, waiter, &s.debugerr)

	waiter.Wait()

	return sout, serr, nil
}

func (s *shell) Exit() {
	s.stdin.Write([]byte("exit" + newline))

	// if it's possible to close stdin, do so (some backends, like the local one,
	// do support it)
	closer, ok := s.stdin.(io.Closer)
	if ok {
		closer.Close()
	}

	s.handle.Wait()

	s.handle = nil
	s.stdin = nil
	s.stdout = nil
	s.stderr = nil
}

func (s *shell) streamReader(stream io.Reader, boundary string, buffer *string, signal *sync.WaitGroup, debug *[]byte) error {
	// read all output until we have found our boundary token
	var lines [][]byte

	scanner := bufio.NewScanner(stream)
	scanner.Split(onBoundary([]byte(boundary)))

	for scanner.Scan() {
		lines = append(lines, scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
		return err
	}

	*buffer = string(bytes.Join(lines, []byte("\n")))
	signal.Done()

	return nil
}

func createBoundary() string {
	return "$gorilla" + utils.CreateRandomString(12) + "$"
}

func onBoundary(boundary []byte) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			if bytes.HasPrefix(data, boundary) {
				return 0, nil, bufio.ErrFinalToken
			}
			return i + 1, data[0:i], nil
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	}
}
