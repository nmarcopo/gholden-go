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
	// TODO maybe we shouldn't worry about things the client will never send
	if msg.Line.Message.ChallstrMessage != nil {
		serializedMsg.WriteString(msg.Line.Message.ChallstrMessage.Challstr)
	}
	if msg.Line.Message.UnknownMessage != nil {
		serializedMsg.WriteString(msg.Line.Message.UnknownMessage.Command)
		serializedMsg.WriteString(Separator)
		serializedMsg.WriteString(msg.Line.Message.UnknownMessage.Data)
	}
	return serializedMsg.String(), nil
}
