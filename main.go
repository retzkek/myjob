package main

import (
	"log"
	"os"

	"github.com/clok/kemba"
	"github.com/urfave/cli/v2"
)

func main() {
	k := kemba.New("myjob")

	k.Log("initializing cli app")
	app := &cli.App{
		Usage: "My job management client",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name:    "status",
				Aliases: []string{},
				Usage:   "fetch submission status",
				Flags:   StatusFlags,
				Action:  Status,
			},
		},
	}

	k.Log("running cli app")
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
