package grammar

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var ShowdownParser = parser{
	parser: participle.MustBuild[ServerMessage](
		participle.Lexer(
			lexer.MustSimple([]lexer.SimpleRule{
				{Name: `EOL`, Pattern: `\n|\r\n`},
				{Name: `Sep`, Pattern: `\` + Separator},
				{Name: `Room`, Pattern: `>`},
				{Name: `Ident`, Pattern: `[a-z]+`},
				{Name: `String`, Pattern: `[^\n]+`},
				{Name: `Whitespace`, Pattern: `[ \t]+`},
			}),
		),
		participle.Elide("Whitespace"),
	),
	debug: testing.Testing(),
}

type parser struct {
	parser *participle.Parser[ServerMessage]
	debug  bool
}

type parserErr struct {
	msg   []byte
	error participle.Error
}

func newParserErr(msg []byte, err error) error {
	return &parserErr{
		msg:   msg,
		error: participle.Wrapf(lexer.Position{}, err, "unable to parse message from showdown"),
	}
}

func (e *parserErr) Error() string {
	return e.error.Error()
}

func Pretty(err error) string {
	if err == nil {
		return ""
	}
	var pErr *parserErr
	if !errors.As(err, &pErr) {
		return err.Error()
	}

	s := strings.Builder{}
	lines := bytes.Split(pErr.msg, []byte("\n"))
	const prefix = "> "
	for l := range pErr.error.Position().Line {
		s.WriteString(prefix)
		s.Write(lines[l])
		s.WriteString("\n")
	}
	s.WriteString(
		fmt.Sprintf(
			"%s%s^",
			strings.Repeat(" ", len(prefix)),
			strings.Repeat(".", pErr.error.Position().Column-1), // nth position should be what we point to
		),
	)
	return s.String()
}

func (p *parser) Parse(bytes []byte) (ServerMessage, error) {
	var opts []participle.ParseOption
	if p.debug {
		opts = append(opts, participle.Trace(os.Stdout))
	}
	val, err := p.parser.ParseBytes("", bytes, opts...)
	if err != nil {
		return ServerMessage{}, newParserErr(bytes, err)
	}
	if val == nil {
		return ServerMessage{}, newParserErr(bytes, errors.New("message was nil"))
	}
	return *val, nil
}
