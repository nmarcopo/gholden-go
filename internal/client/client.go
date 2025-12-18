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
		opts := &slog.HandlerOptions{}
		if c.Debug {
			opts.Level = slog.LevelDebug
		}
		c.Logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	// Initialize websocket connection
	dialCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	conn, _, err := websocket.Dial(dialCtx, c.Address, nil /* opts */)
	if err != nil {
		c.Logger.ErrorContext(dialCtx, "failed to connect to server", "error", errors.WithStack(err))
	}

	// Listen for and log incoming messages from the websocket
	incomingMessages := make(chan grammar.ServerMessage)
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
	wg.Go(func() {
		for {
			select {
			case m := <-incomingMessages:
				c.Logger.InfoContext(subCtx, "Received message", "message", m)
			case <-subCtx.Done():
				c.Logger.InfoContext(subCtx, "context done", "error", subCtx.Err())
				return
			}
		}
	})

	// Send some messages to the server
	outgoingMessages := make(chan grammar.ClientMessage)
	p := newPublisher(outgoingMessages, c.Timeout, c.Logger)
	pubCtx, pubCancel := context.WithCancel(ctx)
	defer pubCancel()
	wg.Go(func() {
		err := p.run(pubCtx, conn)
		pubCancel()
		if err != nil {
			c.Logger.ErrorContext(ctx, "error running publisher", "error", err)
		}
	})
	select {
	case outgoingMessages <- grammar.ClientMessage{
		Line: &grammar.Line{
			RoomID: &grammar.RoomID{
				Room: "lobby",
			},
			Message: &grammar.Message{
				UnknownMessage: &grammar.UnknownMessage{
					Command: "testing123",
					Data:    "",
				},
			},
		},
	}:
	case <-subCtx.Done():
	}

	wg.Wait()

	return nil
}
