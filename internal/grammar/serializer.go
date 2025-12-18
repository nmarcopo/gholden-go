package grammar

import (
	"strings"

	"github.com/pkg/errors"
)

func Serialize(msg ClientMessage) (string, error) {
	if msg.Line == nil {
		return "", errors.New("no line in message")
	}
	var serializedMsg strings.Builder
	if msg.Line.RoomID != nil {
		serializedMsg.WriteString(msg.Line.RoomID.Room)
	}
	serializedMsg.WriteString(Separator)
	if msg.Line.Message == nil {
		return "", errors.New("message should not be nil")
	}
	//serializedMsg.WriteString(msg.Line.Message.Command)
	return serializedMsg.String(), nil
}
