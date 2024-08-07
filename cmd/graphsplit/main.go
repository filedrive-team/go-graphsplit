package main

import (
	"context"
	"fmt"
	"os"

	"github.com/filedrive-team/go-graphsplit"
	"github.com/filedrive-team/go-graphsplit/dataset"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("graphsplit")

func main() {
	logging.SetLogLevel("*", "INFO")
	local := []*cli.Command{
		chunkCmd,
		restoreCmd,
		commpCmd,
		importDatasetCmd,
	}

	app := &cli.App{
		Name:     "graphsplit",
		Flags:    []cli.Flag{},
		Commands: local,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

var chunkCmd = &cli.Command{
	Name:  "chunk",
	Usage: "Generate CAR files of the specified size",
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "slice-size",
			Value: 17179869184, // 16G
			Usage: "specify chunk piece size",
		},
		&cli.UintFlag{
			Name:  "parallel",
			Value: 2,
			Usage: "specify how many number of goroutines runs when generate file node",
		},
		&cli.StringFlag{
			Name:     "graph-name",
			Required: true,
			Usage:    "specify graph name",
		},
		&cli.StringFlag{
			Name:     "car-dir",
			Required: true,
			Usage:    "specify output CAR directory",
		},
		&cli.StringFlag{
			Name:  "parent-path",
			Value: "",
			Usage: "specify graph parent path",
		},
		&cli.BoolFlag{
			Name:  "save-manifest",
			Value: true,
			Usage: "create a mainfest.csv in car-dir to save mapping of data-cids and slice names",
		},
		&cli.BoolFlag{
			Name:  "calc-commp",
			Value: false,
			Usage: "create a mainfest.csv in car-dir to save mapping of data-cids, slice names, piece-cids and piece-sizes",
		},
		&cli.BoolFlag{
			Name:  "rename",
			Value: false,
			Usage: "rename carfile to piece",
		},
		&cli.BoolFlag{
			Name:  "add-padding",
			Value: false,
			Usage: "add padding to carfile in order to convert it to piece file",
		},
	},
	ArgsUsage: "<input path>",
	Action: func(c *cli.Context) error {
		ctx := context.Background()
		parallel := c.Uint("parallel")
		sliceSize := c.Uint64("slice-size")
		parentPath := c.String("parent-path")
		carDir := c.String("car-dir")
		graphName := c.String("graph-name")
		if !graphsplit.ExistDir(carDir) {
			return fmt.Errorf("the path of car-dir does not exist")
		}
		if sliceSize == 0 {
			return fmt.Errorf("slice size has been set as 0")
		}

		targetPath := c.Args().First()
		var cb graphsplit.GraphBuildCallback
		if c.Bool("calc-commp") {
			cb = graphsplit.CommPCallback(carDir, c.Bool("rename"), c.Bool("add-padding"))
		} else if c.Bool("save-manifest") {
			cb = graphsplit.CSVCallback(carDir)
		} else {
			cb = graphsplit.ErrCallback()
		}

		return graphsplit.Chunk(ctx, int64(sliceSize), parentPath, targetPath, carDir, graphName, int(parallel), cb)
	},
}

var restoreCmd = &cli.Command{
	Name:  "restore",
	Usage: "Restore files from CAR files",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "car-path",
			Required: true,
			Usage:    "specify source car path, directory or file",
		},
		&cli.StringFlag{
			Name:     "output-dir",
			Required: true,
			Usage:    "specify output directory",
		},
		&cli.IntFlag{
			Name:  "parallel",
			Value: 4,
			Usage: "specify how many number of goroutines runs when generate file node",
		},
	},
	Action: func(c *cli.Context) error {
		parallel := c.Int("parallel")
		outputDir := c.String("output-dir")
		carPath := c.String("car-path")
		if parallel <= 0 {
			return fmt.Errorf("Unexpected! Parallel has to be greater than 0")
		}

		graphsplit.CarTo(carPath, outputDir, parallel)
		graphsplit.Merge(outputDir, parallel)

		fmt.Println("completed!")
		return nil
	},
}

var commpCmd = &cli.Command{
	Name:  "commP",
	Usage: "PieceCID and PieceSize calculation",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "rename",
			Value: false,
			Usage: "rename carfile to piece",
		},
		&cli.BoolFlag{
			Name:  "add-padding",
			Value: false,
			Usage: "add padding to carfile in order to convert it to piece file",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := context.Background()
		targetPath := c.Args().First()

		res, err := graphsplit.CalcCommP(ctx, targetPath, c.Bool("rename"), c.Bool("add-padding"))
		if err != nil {
			return err
		}

		fmt.Printf("PieceCID: %s, PieceSize: %d\n", res.Root, res.Size)
		return nil
	},
}

var importDatasetCmd = &cli.Command{
	Name:  "import-dataset",
	Usage: "import files from the specified dataset",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "dsmongo",
			Required: true,
			Usage:    "specify the mongodb connection",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := context.Background()

		targetPath := c.Args().First()
		if !graphsplit.ExistDir(targetPath) {
			return fmt.Errorf("Unexpected! The path to dataset does not exist")
		}

		return dataset.Import(ctx, targetPath, c.String("dsmongo"))
	},
}
