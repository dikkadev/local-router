package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"

	"local-router/internal/cli"
	"local-router/internal/router"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		addr := "127.0.0.1:80"
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--help", "-h", "help":
				os.Exit(cli.Run([]string{"serve", "--help"}, cli.Config{}))
			case "--addr":
				i++
				if i >= len(args) {
					fmt.Fprintln(os.Stderr, "--addr requires a value")
					os.Exit(2)
				}
				addr = args[i]
			default:
				fmt.Fprintf(os.Stderr, "unknown serve option %s\n\n", args[i])
				os.Exit(cli.Run([]string{"serve", "--help"}, cli.Config{}))
			}
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := router.NewServer().ListenAndServe(ctx, addr); err != nil {
			log.Error("failed to start local-router", "error", err)
			fmt.Fprintf(os.Stderr, "local-router could not bind %s. Port 80 may require privileges or may already be in use. %v\n", addr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Run(os.Args[1:], cli.Config{}))
}
