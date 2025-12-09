package client

import (
	"context"
	"errors"
	"fmt"
	"gholden-go/internal/parser"
	"time"

	"github.com/alecthomas/repr"
	"github.com/coder/websocket"
)

type CLI struct {
	Address string        `help:"Address to bind to" default:"ws://localhost:8000/showdown/websocket"`
	Timeout time.Duration `help:"Timeout for WebSocket connections" default:"30s"`
	Debug   bool          `help:"Enable debug mode"`
}

func (c *CLI) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, c.Address, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	err = c.listen(conn)
	return err
}

func (c *CLI) listen(conn *websocket.Conn) error {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		t, msg, err := conn.Read(ctx)
		cancel()
		if err != nil {
			return err
		}
		if t == websocket.MessageBinary {
			return errors.New("websocket: unexpected binary message")
		}
		parsed, pErr := parser.ShowdownParser.Parse(msg)
		if pErr != nil {
			fmt.Println(parser.Pretty(pErr))
			return pErr
		}
		repr.Println(msg)
		repr.Println(parsed)
	}
}
