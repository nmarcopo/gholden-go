package client

import (
	"context"
	"log/slog"
	"time"

	"gholden-go/internal/grammar"

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
		msgType, msg, err := conn.Read(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
		p.logger.DebugContext(ctx, "message received", "type", msgType, "body", string(msg))
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
