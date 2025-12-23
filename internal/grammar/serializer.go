package grammar

import (
	"fmt"
	"strings"
)

// TODO maybe this should go in a different package
type ClientMessage interface {
	Serialize() string
}

type Rename struct {
	Username  string
	Assertion string
}

func (r Rename) Serialize() string {
	return fmt.Sprintf("|/trn %s,0,%s", r.Username, r.Assertion)
}

type Help struct {
	// Command can be left empty to see the general help menu
	Command string
}

func (h Help) Serialize() string {
	return fmt.Sprintf("|/help %s", h.Command)
}

type Challenge struct {
	User   string
	Format string
}

func (c Challenge) Serialize() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("|/challenge %s", c.User))
	if c.Format != "" {
		b.WriteString(fmt.Sprintf(", %s", c.Format))
	}
	return b.String()
}

type RawCommand struct {
	Command string
}

func (r RawCommand) Serialize() string {
	return r.Command
}
