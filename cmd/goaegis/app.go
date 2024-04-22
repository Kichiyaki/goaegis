package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"slices"

	"github.com/urfave/cli/v2"
)

const aegisVaultFileName = ".aegis_vault.json"

var (
	appFlagPath = &cli.PathFlag{
		Name:        "path",
		Aliases:     []string{"p"},
		Usage:       "path to vault file",
		Required:    false,
		DefaultText: "$HOME/" + aegisVaultFileName,
	}
	appFlags = []cli.Flag{appFlagPath}
)

type appWrapper struct {
	*cli.App
	logger *slog.Logger
}

func newApp(name, version string) *appWrapper {
	app := &appWrapper{App: cli.NewApp(), logger: slog.Default()}
	app.Name = name
	app.HelpName = name
	app.Version = version
	app.Usage = "Two-Factor Authentication (2FA) App compatible with Aegis vault format"
	app.Commands = []*cli.Command{cmdHook, cmdTUI}
	app.DefaultCommand = cmdTUI.Name
	app.EnableBashCompletion = true
	app.Flags = slices.Concat(appFlags, logFlags)
	app.Before = app.handleBefore
	return app
}

func (a *appWrapper) handleBefore(c *cli.Context) error {
	a.logger = newLoggerFromFlags(c)

	c.Context = loggerToCtx(c.Context, a.logger)

	a.logger.Debug("executing command", slog.Any("args", c.Args().Slice()))

	return nil
}

func getVaultPath(c *cli.Context) (string, error) {
	if p := c.String(appFlagPath.Name); p != "" {
		return p, nil
	}

	dirname, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("couldn't get user home dir: %w", err)
	}

	return path.Join(dirname, aegisVaultFileName), nil
}
