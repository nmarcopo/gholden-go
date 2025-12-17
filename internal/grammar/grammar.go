package grammar

type Showdown struct {
	Lines []*Line `(@@ EOL?)+`
}

type Line struct {
	RoomID  *RoomID  `@@?`
	Message *Message `@@`
}

type RoomID struct {
	Room string `Room @Ident`
}

type Message struct {
	Command string `Sep @Ident Sep`
	Args    string `@String`
}
