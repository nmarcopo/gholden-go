package main

import (
	"context"
	"fmt"
	"gholden-go/internal/client"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli := kong.Parse(&client.CLI{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
	}, kong.BindTo(ctx, (*context.Context)(nil)))
	if err := cli.Run(ctx); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}
