package main

import (
	"fmt"
	"gholden-go/internal/client"
	"os"

	"github.com/alecthomas/kong"
)

func main() {
	ctx := kong.Parse(&client.CLI{})
	if err := ctx.Run(); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
