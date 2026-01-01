package client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"gholden-go/internal/grammar"

	"github.com/coder/websocket"
	"github.com/pkg/errors"
)

type CLI struct {
	Address       string        `help:"Address to bind to" default:"ws://localhost:8000/showdown/websocket"`
	LoginEndpoint string        `help:"Address that serves login" default:"https://play.pokemonshowdown.com/api/login"`
	Timeout       time.Duration `help:"Timeout for individual dials/reads/writes/etc" default:"30s"`
	Debug         bool          `help:"Enable debug mode"`
	Logger        *slog.Logger  `kong:"-"`
	Stdin         io.Reader     `kong:"-"` // required
	Stdout        io.Writer     `kong:"-"` // required
}

func (c *CLI) Run(ctx context.Context) error {
	if c.Logger == nil {
		opts := &slog.HandlerOptions{
			// Include stacktrace in error logs
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				switch a.Key {
				case "error":
					if e, ok := a.Value.Any().(error); ok {
						a = slog.Group(
							a.Key,
							slog.String("error", e.Error()),
							slog.String("errorVerbose", fmt.Sprintf("%+v", e)),
						)
					}
				}
				return a
			},
		}
		if c.Debug {
			opts.Level = slog.LevelDebug
		}
		c.Logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	// Initialize websocket connection
	conn, _, err := websocket.Dial(ctx, c.Address, nil /* opts */)
	if err != nil {
		return errors.WithStack(err)
	}

	// Listen for and log incoming messages from the websocket
	incomingMessages := make(chan grammar.ServerMessage)
	s := newSubscriber(incomingMessages, c.Logger, c.Timeout)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err := s.run(ctx, conn)
		return errors.WithMessage(err, "error running subscriber")
	})

	// Send some messages to the server
	outgoingMessages := make(chan grammar.ClientMessage)
	p := newPublisher(outgoingMessages, c.Timeout, c.Logger)
	g.Go(func() error {
		err := p.run(ctx, conn)
		return errors.WithMessage(err, "error running publisher")
	})

	controller := newController(controllerOpts{
		outgoingMessagesCh: outgoingMessages,
		incomingMessagesCh: incomingMessages,
		loginEndpoint:      c.LoginEndpoint,
		timeout:            c.Timeout,
		logger:             c.Logger,
		stdin:              c.Stdin,
		stdout:             c.Stdout,
	})
	g.Go(func() error {
		err := controller.handleIncoming(ctx)
		return errors.WithMessage(err, "error running controller")
	})
	g.Go(func() error {
		err := controller.prompt(ctx)
		return errors.WithMessage(err, "error running prompt")
	})

	return g.Wait()
}
