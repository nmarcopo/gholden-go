package client

import (
	"context"
	"gholden-go/internal/grammar"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/pkg/errors"
)

type subscriber struct {
	queue   chan<- *grammar.Showdown
	logger  *slog.Logger
	timeout time.Duration
}

func newSubscriber(queue chan<- *grammar.Showdown, logger *slog.Logger, timeout time.Duration) *subscriber {
	return &subscriber{
		queue:   queue,
		logger:  logger,
		timeout: timeout,
	}
}

func (p *subscriber) run(ctx context.Context, conn *websocket.Conn) error {
	for {
		msgType, msg, err := conn.Read(ctx)
		if err != nil {
			p.logger.WarnContext(ctx, "websocket read error", "error", errors.WithStack(err))
			continue
		}
		if msgType != websocket.MessageText {
			p.logger.WarnContext(ctx, "websocket message type not supported", "message type", msgType)
			continue
		}

		parsed, err := grammar.ShowdownParser.Parse(msg)
		if err != nil {
			p.logger.WarnContext(ctx, "message parse error", "error", err)
			continue
		}
		select {
		case p.queue <- parsed:
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		}
	}
}
