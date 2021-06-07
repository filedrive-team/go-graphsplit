package graphsplit

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("graphsplit")

type GraphBuildCallback interface {
	OnSuccess(node ipld.Node, graphName, fsDetail string)
	OnError(error)
}

type commPCallback struct {
	carDir string
}

func (cc *commPCallback) OnSuccess(node ipld.Node, graphName, fsDetail string) {
	commpStartTime := time.Now()
	carfilepath := path.Join(cc.carDir, node.Cid().String()+".car")
	cpRes, err := CalcCommP(context.TODO(), carfilepath)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("calculation of pieceCID completed, time elapsed: %s", time.Now().Sub(commpStartTime))
	// Add node inof to manifest.csv
	manifestPath := path.Join(cc.carDir, "manifest.csv")
	_, err = os.Stat(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
	var isCreateAction bool
	if err != nil && os.IsNotExist(err) {
		isCreateAction = true
	}
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if isCreateAction {
		if _, err := f.Write([]byte("playload_cid,filename,piece_cid,piece_size,detail\n")); err != nil {
			log.Fatal(err)
		}
	}
	if _, err := f.Write([]byte(fmt.Sprintf("%s,%s,%s,%d,%s\n", node.Cid(), graphName, cpRes.Root.String(), cpRes.Size, fsDetail))); err != nil {
		log.Fatal(err)
	}
}

func (cc *commPCallback) OnError(err error) {
	log.Fatal(err)
}

type csvCallback struct {
	carDir string
}

func (cc *csvCallback) OnSuccess(node ipld.Node, graphName, fsDetail string) {
	// Add node inof to manifest.csv
	manifestPath := path.Join(cc.carDir, "manifest.csv")
	_, err := os.Stat(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}
	var isCreateAction bool
	if err != nil && os.IsNotExist(err) {
		isCreateAction = true
	}
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if isCreateAction {
		if _, err := f.Write([]byte("playload_cid,filename,detail\n")); err != nil {
			log.Fatal(err)
		}
	}
	if _, err := f.Write([]byte(fmt.Sprintf("%s,%s,%s\n", node.Cid(), graphName, fsDetail))); err != nil {
		log.Fatal(err)
	}
}

func (cc *csvCallback) OnError(err error) {
	log.Fatal(err)
}

type errCallback struct{}

func (cc *errCallback) OnSuccess(ipld.Node, string, string) {}
func (cc *errCallback) OnError(err error) {
	log.Fatal(err)
}

func CommPCallback(carDir string) GraphBuildCallback {
	return &commPCallback{carDir: carDir}
}

func CSVCallback(carDir string) GraphBuildCallback {
	return &csvCallback{carDir: carDir}
}
func ErrCallback() GraphBuildCallback {
	return &errCallback{}
}

func Chunk(ctx context.Context, sliceSize int64, parentPath, targetPath, carDir, graphName string, parallel int, cb GraphBuildCallback) error {
	var cumuSize int64 = 0
	graphSliceCount := 0
	graphFiles := make([]Finfo, 0)
	if sliceSize == 0 {
		return xerrors.Errorf("Unexpected! Slice size has been set as 0")
	}
	if parallel <= 0 {
		return xerrors.Errorf("Unexpected! Parallel has to be greater than 0")
	}
	if parentPath == "" {
		parentPath = targetPath
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
			BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel, cb)
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
			BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel, cb)
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
					BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel, cb)
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
		BuildIpldGraph(ctx, graphFiles, GenGraphName(graphName, graphSliceCount, sliceTotal), parentPath, carDir, parallel, cb)
		fmt.Printf("cumu-size: %d\n", cumuSize)
		fmt.Printf(GenGraphName(graphName, graphSliceCount, sliceTotal))
		fmt.Printf("=================\n")
	}
	return nil
}
