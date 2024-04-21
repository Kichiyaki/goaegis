package main

import (
	internal2 "goaegis/internal"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v2"
)

var cmdTUI = &cli.Command{
	Name: "tui",
	Action: func(c *cli.Context) error {
		logger := loggerFromCtx(c.Context)

		path, err := getVaultPath(c)
		if err != nil {
			return err
		}

		logger.Debug("trying to read vault file...", slog.String("path", path))

		vault, err := internal2.NewVaultFromFile(path)
		if err != nil {
			return err
		}

		logger.Debug("vault file read successfully", slog.String("path", path))

		if _, err := tea.NewProgram(internal2.NewUI(vault)).Run(); err != nil {
			return err
		}

		return nil
	},
}
