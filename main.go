package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"

	"forge.dikka.dev/lab/local-router/internal/cli"
	"forge.dikka.dev/lab/local-router/internal/router"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		addr := envOrDefault("LOCAL_ROUTER_ADDR", "127.0.0.1:80")
		externalHost := os.Getenv("LOCAL_ROUTER_EXTERNAL_HOST")
		jsonLogs := false
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--help", "-h", "help":
				os.Exit(cli.Run([]string{"serve", "--help"}, cli.Config{}))
			case "--json":
				jsonLogs = true
			case "--addr":
				i++
				if i >= len(args) {
					fmt.Fprintln(os.Stderr, "--addr requires a value")
					os.Exit(2)
				}
				addr = args[i]
			case "--external-host":
				i++
				if i >= len(args) {
					fmt.Fprintln(os.Stderr, "--external-host requires a value")
					os.Exit(2)
				}
				externalHost = args[i]
			default:
				fmt.Fprintf(os.Stderr, "unknown serve option %s\n\n", args[i])
				os.Exit(cli.Run([]string{"serve", "--help"}, cli.Config{}))
			}
		}
		if jsonLogs {
			log.SetFormatter(log.JSONFormatter)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		server := router.NewServer()
		server.ExternalHost = externalHost
		if err := server.ListenAndServe(ctx, addr); err != nil {
			log.Error("failed to start local-router", "error", err)
			fmt.Fprintf(os.Stderr, "local-router could not bind %s. Port 80 may require privileges or may already be in use. %v\n", addr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(cli.Run(os.Args[1:], cli.Config{}))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
