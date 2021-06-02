package main

import (
	"context"
	"fmt"
	"os"

	"github.com/filedrive-team/go-graphsplit/piece"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("commp")

func main() {
	logging.SetLogLevel("*", "INFO")
	local := []*cli.Command{
		singleCmd,
	}

	app := &cli.App{
		Name:     "commp",
		Flags:    []cli.Flag{},
		Commands: local,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

var singleCmd = &cli.Command{
	Name:  "single",
	Usage: "PieceCID and PieceSize calculation",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		ctx := context.Background()
		targetPath := c.Args().First()

		res, err := piece.CalcCommP(ctx, targetPath)
		if err != nil {
			return err
		}

		fmt.Printf("PieceCID: %s, PieceSize: %d\n", res.Root, res.Size)
		return nil
	},
}
