package client

import (
	"context"
	"gholden-go/internal/grammar"
	"log/slog"
	"strings"
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

func (p *publisher) Run(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {
		case msg := <-p.queue:
			if msg.Line == nil {
				return errors.New("no line in message")
			}
			var serializedMsg strings.Builder
			if msg.Line.RoomID != nil {
				serializedMsg.WriteString(msg.Line.RoomID.Room)
			}
			serializedMsg.WriteString(grammar.Separator)
			if msg.Line.Message == nil {
				return errors.New("message should not be nil")
			}
			serializedMsg.WriteString(msg.Line.Message.Command)
			writeCtx, cancel := context.WithTimeout(ctx, p.timeout)
			err := conn.Write(writeCtx, websocket.MessageText, []byte(serializedMsg.String()))
			if err != nil {
				p.logger.WarnContext(writeCtx, "failed to write serialized message", "error", errors.WithStack(err))
			}
			cancel()
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		}
	}
}
