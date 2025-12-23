package client

import (
	"errors"
)

type state struct {
	challstr   string
	challstrCh chan struct{}
}

func (s *state) setChallstr(challstr string) error {
	if s.challstr != "" {
		return errors.New("challstr already set")
	}
	s.challstr = challstr
	close(s.challstrCh)
	return nil
}
