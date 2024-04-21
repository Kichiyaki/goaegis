package main

import (
	"log/slog"
	"os"
)

const appName = "goaegis"

// this flag will be set by the build flags
var version = "development"

func main() {
	app := newApp(appName, version)
	if err := app.Run(os.Args); err != nil {
		app.logger.Error("app run failed", slog.Any("error", err))
		os.Exit(1)
	}
}
