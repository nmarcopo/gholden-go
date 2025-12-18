package client

import (
	"context"
	"gholden-go/internal/grammar"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/pkg/errors"
)

type publisher struct {
	queue   <-chan grammar.ClientMessage
	timeout time.Duration
	logger  *slog.Logger
}

func newPublisher(queue <-chan grammar.ClientMessage, timeout time.Duration, logger *slog.Logger) *publisher {
	return &publisher{
		queue:   queue,
		timeout: timeout,
		logger:  logger,
	}
}

func (p *publisher) run(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case msg := <-p.queue:
			message, err := grammar.Serialize(msg)
			if err != nil {
				p.logger.WarnContext(ctx, "failed to serialize message", "error", err)
				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, p.timeout)
			err = conn.Write(writeCtx, websocket.MessageText, []byte(message))
			if err != nil {
				p.logger.WarnContext(writeCtx, "failed to write serialized message", "error", errors.WithStack(err))
			}
			cancel()
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		}
	}
}
