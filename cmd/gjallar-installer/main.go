package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/app"
)

func main() {
	// Unwind on Ctrl-C, a dropped SSH session, or SIGTERM so the installer's
	// deferred cleanup runs: without it the target bind mounts over /boot
	// and /var/lib stayed on the live host (e2e-target, 2026-09-29). The
	// first signal restores default handling, so a second one still kills
	// an installer blocked on a terminal prompt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-ctx.Done()
		stop()
	}()
	os.Exit(app.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
