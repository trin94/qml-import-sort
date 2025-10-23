// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/trin94/qml-import-sort/internal"
	"github.com/urfave/cli/v3"
)

var customHelpText = `USAGE:
   qmlimportsort [flags] [files...]
{{if .VisibleFlags}}
FLAGS:{{template "visibleFlagTemplate" .}}{{end}}
`

func main() {
	log.SetFlags(0)

	err := command().Run(context.Background(), os.Args)

	if err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}
}

func command() *cli.Command {
	cli.RootCommandHelpTemplate = customHelpText
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("qmlimportsort %s\n", cmd.Root().Version)
	}

	return &cli.Command{
		Name:    "qmlimportsort",
		Version: "v0.1.0",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "in-place",
				Aliases: []string{"i"},
				Usage:   "modify files in-place (only valid with files)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			files := cmd.Args().Slice()
			inPlace := cmd.Bool("in-place")
			if inPlace && len(files) == 0 {
				return cli.Exit("error: --in-place can only be used with files", 1)
			}
			if len(files) > 0 {
				return internal.ProcessFiles(files, inPlace)
			} else {
				return internal.ProcessStdIn(os.Stdin)
			}
		},
	}
}
