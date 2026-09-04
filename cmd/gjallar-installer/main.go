package main

import (
	"context"
	"os"

	"github.com/bakanura/gjallarOS/internal/installer/app"
)

func main() { os.Exit(app.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
