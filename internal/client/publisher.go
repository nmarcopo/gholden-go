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
			p.logger.DebugContext(ctx, "sending message", "message", msg)
			writeCtx, cancel := context.WithTimeout(ctx, p.timeout)
			err := conn.Write(writeCtx, websocket.MessageText, []byte(msg.Serialize()))
			if err != nil {
				p.logger.WarnContext(writeCtx, "failed to write serialized message", "error", errors.WithStack(err))
			}
			cancel()
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		}
	}
}
