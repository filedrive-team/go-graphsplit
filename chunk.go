package graphsplit

import (
	"context"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("graphsplit")

func Chunk(ctx context.Context, sliceSize int64, targetPath, carDir, graphName string, parallel int) error {
	var cumuSize int64 = 0
	graphSliceCount := 0
	graphFiles := make([]Finfo, 0)
	if sliceSize == 0 {
		return xerrors.Errorf("Unexpected! Slice size has been set as 0")
	}
	if parallel <= 0 {
		return xerrors.Errorf("Unexpected! Parallel has to be greater than 0")
	}

	args := []string{targetPath}
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
			BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), targetPath, carDir, parallel)
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
			BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), targetPath, carDir, parallel)
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
					BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), targetPath, carDir, parallel)
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
		BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), targetPath, carDir, parallel)
		fmt.Printf("cumu-size: %d\n", cumuSize)
		fmt.Printf(GenGraphName(graphName, graphSliceCount, sliceTotal))
		fmt.Printf("=================\n")
	}
	return nil
}
