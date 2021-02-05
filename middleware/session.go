// Copyright (c) 2017 Gorillalabs. All rights reserved.

package middleware

import (
	"fmt"
	"strings"

	"github.com/benmoss/go-powershell/utils"
)

type session struct {
	upstream Middleware
	name     string
}

func NewSession(upstream Middleware, config *SessionConfig) (Middleware, error) {
	asserted, ok := config.Credential.(credential)
	if ok {
		credentialParamValue, err := asserted.prepare(upstream)
		if err != nil {
			return nil, fmt.Errorf("could not setup credentials: %v", err)
		}

		config.Credential = credentialParamValue
	}

	name := "goSess" + utils.CreateRandomString(8)
	args := strings.Join(config.ToArgs(), " ")

	_, _, err := upstream.Execute(fmt.Sprintf("$%s = New-PSSession %s", name, args))
	if err != nil {
		return nil, fmt.Errorf("could not create new PSSession: %v", err)
	}

	return &session{upstream, name}, nil
}

func (s *session) Execute(cmd string) (string, string, error) {
	return s.upstream.Execute(fmt.Sprintf("Invoke-Command -Session $%s -Script {%s}", s.name, cmd))
}

func (s *session) Exit() {
	s.upstream.Execute(fmt.Sprintf("Disconnect-PSSession -Session $%s", s.name))
	s.upstream.Exit()
}
