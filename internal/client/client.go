package client

import (
	"context"
	"gholden-go/internal/grammar"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/pkg/errors"
)

type CLI struct {
	Address string        `help:"Address to bind to" default:"ws://localhost:8000/showdown/websocket"`
	Timeout time.Duration `help:"Timeout for WebSocket dials/reads/writes" default:"30s"`
	Debug   bool          `help:"Enable debug mode"`
	Logger  *slog.Logger  `kong:"-"`
}

func (c *CLI) Run() error {
	ctx := context.Background()
	if c.Logger == nil {
		c.Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil /* opts */))
	}

	// Initialize websocket connection
	dialCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	conn, _, err := websocket.Dial(dialCtx, c.Address, nil /* opts */)
	if err != nil {
		c.Logger.ErrorContext(dialCtx, "failed to connect to server", "error", errors.WithStack(err))
	}

	incomingMessages := make(chan *grammar.Showdown)
	s := newSubscriber(incomingMessages, c.Logger, c.Timeout)
	wg := &sync.WaitGroup{}
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()
	wg.Go(func() {
		err := s.run(subCtx, conn)
		subCancel()
		if err != nil {
			c.Logger.ErrorContext(ctx, "error running subscriber", "error", err)
		}
	})
	for {
		select {
		case m := <-incomingMessages:
			c.Logger.InfoContext(subCtx, "Received message", "message", m)
		case <-subCtx.Done():
			return errors.WithStack(subCtx.Err())
		}
	}
}
