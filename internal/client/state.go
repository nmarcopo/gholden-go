package client

import (
	"errors"
	"sync"
)

type challstr struct {
	mu       sync.Mutex
	challstr string
	set      chan struct{}
}

type state struct {
	challstr challstr
}

func (s *state) setChallstr(challstr string) error {
	s.challstr.mu.Lock()
	defer s.challstr.mu.Unlock()
	if s.challstr.challstr != "" {
		return errors.New("challstr already set")
	}
	s.challstr.challstr = challstr
	close(s.challstr.set)
	return nil
}
