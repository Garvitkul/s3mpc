package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Garvitkul/s3mpc/internal/app"
)

// Version information (set by build flags)
var (
	Version   = "1.0.2"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

func main() {
	ctx := context.Background()
	
	app := app.NewApp(Version)
	if err := app.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}