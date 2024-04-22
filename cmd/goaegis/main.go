package main

import (
	"log/slog"
	"os"
	"runtime/debug"
	"slices"
)

const appName = "goaegis"

const defaultVersion = "development"

// this flag will be set by the build flags
var version = defaultVersion

func main() {
	app := newApp(appName, getVersion())
	if err := app.Run(os.Args); err != nil {
		app.logger.Error("app run failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func getVersion() string {
	if version != defaultVersion {
		return version
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok || slices.Contains([]string{"", "(devel)"}, buildInfo.Main.Version) {
		return defaultVersion
	}

	return buildInfo.Main.Version
}
