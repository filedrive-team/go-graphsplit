package main

import (
	"context"
	"fmt"
	logging "github.com/ipfs/go-log/v2"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"os"
)

var log = logging.Logger("graphsplit")

func main() {
	logging.SetLogLevel("*", "INFO")
	local := []*cli.Command{
		chunkCmd,
		restoreCmd,
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
		&cli.Int64Flag{
			Name:  "slice-size",
			Value: 17179869184, // 16G
			Usage: "specify chunk piece size",
		},
		&cli.IntFlag{
			Name:  "parallel",
			Value: 4,
			Usage: "specify how many number of goroutines runs when generate file node",
		},
		&cli.StringFlag{
			Name:     "graph-name",
			Required: true,
			Usage:    "specify graph name",
		},
		&cli.StringFlag{
			Name:  "parent-path",
			Value: "",
			Usage: "specify graph parent path",
		},
		&cli.StringFlag{
			Name:     "car-dir",
			Required: true,
			Usage:    "specify output CAR directory",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := context.Background()
		parallel := c.Int("parallel")
		var cumuSize int64 = 0
		sliceSize := c.Int64("slice-size")
		parentPath := c.String("parent-path")
		carDir := c.String("car-dir")
		graphName := c.String("graph-name")
		graphSliceCount := 0
		graphFiles := make([]Finfo, 0)
		if sliceSize == 0 {
			return xerrors.Errorf("Unexpected! Slice size has been set as 0")
		}
		if parallel <= 0 {
			return xerrors.Errorf("Unexpected! Parallel has to be greater than 0")
		}

		args := c.Args().Slice()
		sliceTotal := GetGraphCount(args, sliceSize)
		if sliceTotal == 0 {
			log.Warn("Empty folder or file!")
			return nil
		}
		files := GetFileListAsync(args)
		for item := range files {
			fileSize := item.Info.Size()
			switch {
			case cumuSize+fileSize < sliceSize:
				cumuSize += fileSize
				graphFiles = append(graphFiles, item)
			case cumuSize+fileSize == sliceSize:
				cumuSize += fileSize
				graphFiles = append(graphFiles, item)
				// todo build ipld from graphFiles
				BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel)
				fmt.Printf("cumu-size: %d\n", cumuSize)
				fmt.Printf(GenGraphName(graphName, graphSliceCount, sliceTotal))
				fmt.Printf("=================\n")
				cumuSize = 0
				graphFiles = make([]Finfo, 0)
				graphSliceCount++
			case cumuSize+fileSize > sliceSize:
				fileSliceCount := 0
				// need to split item to fit graph slice
				//
				// first cut
				firstCut := sliceSize - cumuSize
				var seekStart int64 = 0
				var seekEnd int64 = seekStart + firstCut - 1
				fmt.Printf("first cut %d, seek start at %d, end at %d", firstCut, seekStart, seekEnd)
				fmt.Printf("----------------\n")
				graphFiles = append(graphFiles, Finfo{
					Path:      item.Path,
					Name:      fmt.Sprintf("%s.%08d", item.Info.Name(), fileSliceCount),
					Info:      item.Info,
					SeekStart: seekStart,
					SeekEnd:   seekEnd,
				})
				fileSliceCount++
				// todo build ipld from graphFiles
				BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel)
				fmt.Printf("cumu-size: %d\n", cumuSize+firstCut)
				fmt.Printf(GenGraphName(graphName, graphSliceCount, sliceTotal))
				fmt.Printf("=================\n")
				cumuSize = 0
				graphFiles = make([]Finfo, 0)
				graphSliceCount++
				for seekEnd < fileSize-1 {
					seekStart = seekEnd + 1
					seekEnd = seekStart + sliceSize - 1
					if seekEnd >= fileSize-1 {
						seekEnd = fileSize - 1
					}
					fmt.Printf("following cut %d, seek start at %d, end at %d", seekEnd-seekStart+1, seekStart, seekEnd)
					fmt.Printf("----------------\n")
					cumuSize += seekEnd - seekStart + 1
					graphFiles = append(graphFiles, Finfo{
						Path:      item.Path,
						Name:      fmt.Sprintf("%s.%08d", item.Info.Name(), fileSliceCount),
						Info:      item.Info,
						SeekStart: seekStart,
						SeekEnd:   seekEnd,
					})
					fileSliceCount++
					if seekEnd-seekStart == sliceSize-1 {
						// todo build ipld from graphFiles
						BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel)
						fmt.Printf("cumu-size: %d\n", sliceSize)
						fmt.Printf(GenGraphName(graphName, graphSliceCount, sliceTotal))
						fmt.Printf("=================\n")
						cumuSize = 0
						graphFiles = make([]Finfo, 0)
						graphSliceCount++
					}
				}

			}
		}
		if cumuSize > 0 {
			// todo build ipld from graphFiles
			BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel)
			fmt.Printf("cumu-size: %d\n", cumuSize)
			fmt.Printf(GenGraphName(graphName, graphSliceCount, sliceTotal))
			fmt.Printf("=================\n")
		}
		return nil
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
			return xerrors.Errorf("Unexpected! Parallel has to be greater than 0")
		}

		CarTo(carPath, outputDir, parallel)
		Merge(outputDir, parallel)

		fmt.Println("completed!")
		return nil
	},
}
