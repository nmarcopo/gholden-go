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
	queue   chan<- grammar.ServerMessage
	logger  *slog.Logger
	timeout time.Duration
}

func newSubscriber(queue chan<- grammar.ServerMessage, logger *slog.Logger, timeout time.Duration) *subscriber {
	return &subscriber{
		queue:   queue,
		logger:  logger,
		timeout: timeout,
	}
}

// run reads messages from the websocket, parses them into structs, and sends the structs to the queue
func (p *subscriber) run(ctx context.Context, conn *websocket.Conn) error {
	for {
		readCtx, cancel := context.WithTimeout(ctx, p.timeout)
		msgType, msg, err := conn.Read(readCtx)
		cancel()
		if err != nil {
			// TODO send shutdown signal if this happens
			p.logger.ErrorContext(ctx, "websocket read error", "error", errors.WithStack(err))
			return errors.WithStack(err)
		}
		if msgType != websocket.MessageText {
			p.logger.WarnContext(ctx, "websocket message type not supported", "message type", msgType)
			continue
		}

		parsed, err := grammar.ShowdownParser.Parse(msg)
		if err != nil {
			p.logger.WarnContext(ctx, "message parse error", "error", err)
			p.logger.DebugContext(ctx, "detailed error", "error", grammar.Pretty(err))
			continue
		}
		select {
		case p.queue <- parsed:
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		}
	}
}
