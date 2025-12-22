package grammar

import "fmt"

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
