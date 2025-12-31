package client

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
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
	state              *state
	logger             *slog.Logger
	stdin              io.Reader
	stdout             io.Writer
}

type controllerOpts struct {
	outgoingMessagesCh chan<- grammar.ClientMessage // required
	incomingMessagesCh <-chan grammar.ServerMessage // required
	loginEndpoint      string                       // required
	timeout            time.Duration                // required
	logger             *slog.Logger                 // required
	stdin              io.Reader                    // required
	stdout             io.Writer                    // required
}

func newController(opts controllerOpts) *controller {
	return &controller{
		outgoingMessagesCh: opts.outgoingMessagesCh,
		incomingMessagesCh: opts.incomingMessagesCh,
		httpClient: &http.Client{
			Timeout: opts.timeout,
		},
		loginEndpoint: opts.loginEndpoint,
		state: &state{
			challstrCh: make(chan struct{}),
		},
		logger: opts.logger,
		stdin:  opts.stdin,
		stdout: opts.stdout,
	}
}

func (c *controller) handleIncoming(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		case msg := <-c.incomingMessagesCh:
			c.logger.DebugContext(ctx, "Received incoming message", "message", msg)
			for _, line := range msg.Lines {
				if line.Message == nil {
					c.logger.WarnContext(ctx, "Received line without a message", "line", line)
					continue
				}
				switch {
				case line.Message.ChallstrMessage != nil:
					if err := c.state.setChallstr(line.Message.ChallstrMessage.Challstr); err != nil {
						c.logger.WarnContext(ctx, "Error setting challstr value", "error", errors.WithStack(err))
						continue
					}
				default:
					c.logger.DebugContext(ctx, "unsupported message", "message", line)
				}
			}
		}
	}
}

func (c *controller) prompt(ctx context.Context) error {
	// Wait for the challstr to be ready
	select {
	case <-ctx.Done():
		return errors.WithStack(ctx.Err())
	case <-c.state.challstrCh:
	}

	login := loginInput{
		Name: "test" + uuid.New().String()[:12], // generate a random (most likely unused) username for now
		// NB: The password field must exist but doesn't actually matter unless the username is already registered
		Pass:     "1234",
		Challstr: c.state.challstr,
	}
	c.logger.InfoContext(ctx, "logging in", "username", login.Name)
	if err := c.login(ctx, login); err != nil {
		return errors.WithMessage(err, "failed to login")
	}

	c.logger.InfoContext(ctx, "enter commands")
	inputCh := make(chan string)
	scanner := bufio.NewScanner(c.stdin)
	go func() {
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
		close(inputCh)
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		case input, ok := <-inputCh:
			if !ok {
				return errors.WithMessage(cmp.Or(scanner.Err(), io.EOF), "input channel closed")
			}
			c.logger.InfoContext(ctx, "enter input", "input", input)
			c.outgoingMessagesCh <- grammar.RawCommand{Command: input}
		}
	}
}

type loginInput struct {
	Name     string `json:"name"`     // required
	Pass     string `json:"pass"`     // required
	Challstr string `json:"challstr"` // required
}

type loginResponse struct {
	Assertion string `json:"assertion"`
}

// login logs in to Showdown following the guidance in the protocol documentation:
// https://github.com/smogon/pokemon-showdown/blob/master/PROTOCOL.md
func (c *controller) login(ctx context.Context, input loginInput) error {
	// From docs:
	// you'll need to make an HTTP POST request to https://play.pokemonshowdown.com/api/login with the data
	// name=USERNAME&pass=PASSWORD&challstr=CHALLSTR
	// USERNAME is your username and PASSWORD is your password, and CHALLSTR is the value you got from |challstr|.
	// Note that CHALLSTR contains | characters.
	c.logger.DebugContext(ctx, "Sending login request", "username", input.Name, "challstr", input.Challstr)
	body, err := json.Marshal(input)
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
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("login request failed with status %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}
	c.logger.DebugContext(ctx, "response from login", "code", resp.StatusCode, "body", string(b))
	var l loginResponse
	// Body is prefixed by a `]` character, per the docs https://github.com/smogon/pokemon-showdown/blob/master/PROTOCOL.md
	if err := json.Unmarshal(bytes.TrimPrefix(b, []byte("]")), &l); err != nil {
		return errors.Wrap(err, "failed to unmarshal login response")
	}

	// From docs:
	// Finish logging in (or renaming) by sending:
	// /trn USERNAME,0,ASSERTION where USERNAME is your desired username and ASSERTION is data.assertion
	select {
	case c.outgoingMessagesCh <- grammar.Rename{
		Username:  input.Name,
		Assertion: l.Assertion,
	}:
		c.logger.DebugContext(ctx, "sent login command to socket")
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "failed to send login command to socket")
	}
	return nil
}
