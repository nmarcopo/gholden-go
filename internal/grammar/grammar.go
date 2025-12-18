package grammar

const (
	Separator = "|"
)

type ServerMessage struct {
	Lines []*Line `(@@ EOL?)+`
}

// ClientMessage is not added to the parser since we only need to parse messages we receive in plaintext
type ClientMessage struct {
	Line *Line
}

type Line struct {
	RoomID  *RoomID  `@@?`
	Message *Message `@@`
}

type RoomID struct {
	Room string `Room @Ident`
}

type Message struct {
	ChallstrMessage *ChallstrMessage `  @@`
	UnknownMessage  *UnknownMessage  `| @@`
}

type ChallstrMessage struct {
	Command  string `Sep "challstr" Sep`
	Challstr string `@String`
}

type UnknownMessage struct {
	Command string `Sep @Ident Sep`
	Data    string `@String`
}
