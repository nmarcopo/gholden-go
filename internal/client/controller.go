package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gholden-go/internal/grammar"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type controller struct {
	outgoingMessagesCh chan<- grammar.ClientMessage
	incomingMessagesCh <-chan grammar.ServerMessage
	httpClient         *http.Client
	loginEndpoint      string
	logger             *slog.Logger
}

type controllerOpts struct {
	outgoingMessagesCh chan<- grammar.ClientMessage // required
	incomingMessagesCh <-chan grammar.ServerMessage // required
	loginEndpoint      string                       // required
	timeout            time.Duration                // required
	logger             *slog.Logger                 // required
}

func newController(opts controllerOpts) *controller {
	return &controller{
		outgoingMessagesCh: opts.outgoingMessagesCh,
		incomingMessagesCh: opts.incomingMessagesCh,
		httpClient: &http.Client{
			Timeout: opts.timeout,
		},
		loginEndpoint: opts.loginEndpoint,
		logger:        opts.logger,
	}
}

func (c *controller) handleIncoming(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		case msg := <-c.incomingMessagesCh:
			c.logger.InfoContext(ctx, "Received incoming message", "message", msg)
			for _, line := range msg.Lines {
				if line.Message == nil {
					c.logger.WarnContext(ctx, "Received line without a message", "line", line)
					continue
				}
				switch {
				case line.Message.ChallstrMessage != nil:
					if err := c.login(ctx, *line.Message.ChallstrMessage); err != nil {
						c.logger.ErrorContext(ctx, "failed to login", "error", err)
					} else {
						c.logger.InfoContext(ctx, "login successful")
					}
				}
			}
		}
	}
}

type loginInput struct {
	Name     string `json:"name"`
	Pass     string `json:"pass"`
	Challstr string `json:"challstr"`
}

type loginResponse struct {
	Assertion string `json:"assertion"`
}

// login logs in to Showdown following the guidance in the protocol documentation:
// https://github.com/smogon/pokemon-showdown/blob/master/PROTOCOL.md
func (c *controller) login(ctx context.Context, challstr grammar.ChallstrMessage) error {
	// From docs:
	// you'll need to make an HTTP POST request to https://play.pokemonshowdown.com/api/login with the data
	// name=USERNAME&pass=PASSWORD&challstr=CHALLSTR
	// USERNAME is your username and PASSWORD is your password, and CHALLSTR is the value you got from |challstr|.
	// Note that CHALLSTR contains | characters.
	values := loginInput{
		Name: "test" + uuid.New().String()[:12], // generate a random (most likely unused) username for now
		// NB: The password field must exist but doesn't actually matter unless the username is already registered
		Pass:     "1234",
		Challstr: challstr.Challstr,
	}
	c.logger.DebugContext(ctx, "Sending login request", "values", values)
	body, err := json.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "failed to marshal login input")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.loginEndpoint, bytes.NewReader(body))
	if err != nil {
		return errors.Wrap(err, "failed to create login request")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send login request")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.ErrorContext(context.Background(), "failed to close response body", "error", errors.WithStack(err))
		}
	}()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}
	c.logger.DebugContext(ctx, "response from login", "code", resp.StatusCode, "body", string(body))
	var l loginResponse
	// Body is prefixed by a `]` character, per the docs https://github.com/smogon/pokemon-showdown/blob/master/PROTOCOL.md
	if err := json.Unmarshal(bytes.TrimPrefix(b, []byte("]")), &l); err != nil {
		return errors.Wrap(err, "failed to unmarshal login response")
	}

	// From docs:
	// Finish logging in (or renaming) by sending:
	// /trn USERNAME,0,ASSERTION where USERNAME is your desired username and ASSERTION is data.assertion
	select {
	case c.outgoingMessagesCh <- grammar.ClientMessage{
		Line: &grammar.Line{
			Message: &grammar.Message{
				UnknownMessage: &grammar.UnknownMessage{
					// TODO make this easier to interact with
					Data: fmt.Sprintf("/trn %s,0,%s", values.Name, l.Assertion),
				},
			},
		},
	}:
		c.logger.DebugContext(ctx, "sent login command to socket")
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "failed to send login command to socket")
	}
	return nil
}
