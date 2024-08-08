package graphsplit

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("graphsplit")

type GraphBuildCallback interface {
	OnSuccess(buf *Buffer, graphName, payloadCid, fsDetail string)
	OnError(error)
}

type commPCallback struct {
	carDir     string
	rename     bool
	addPadding bool
}

func (cc *commPCallback) OnSuccess(buf *Buffer, graphName, payloadCid, fsDetail string) {
	commpStartTime := time.Now()

	log.Info("start to calculate pieceCID")
	cpRes, err := CalcCommPV2(buf, cc.addPadding)
	if err != nil {
		log.Fatalf("calculation of pieceCID failed: %s", err)
	}
	log.Infof("calculation of pieceCID completed, time elapsed: %s", time.Since(commpStartTime))
	log.Infof("piece cid: %s, payload size: %d, size: %d ", cpRes.Root.String(), cpRes.PayloadSize, cpRes.Size)

	buf.SeekStart()
	if err := os.WriteFile(path.Join(cc.carDir, cpRes.Root.String()+".car"), buf.Bytes(), 0o644); err != nil {
		log.Fatalf("failed to write car file: %s", err)
	}

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
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	csvWriter := csv.NewWriter(f)
	csvWriter.UseCRLF = true
	defer csvWriter.Flush()
	if isCreateAction {
		csvWriter.Write([]string{
			"payload_cid", "filename", "piece_cid", "payload_size", "piece_size", "detail",
		})
	}

	if err := csvWriter.Write([]string{
		payloadCid, graphName, cpRes.Root.String(),
		strconv.FormatInt(cpRes.PayloadSize, 10), strconv.FormatUint(uint64(cpRes.Size), 10), fsDetail,
	}); err != nil {
		log.Fatal(err)
	}
}

func (cc *commPCallback) OnError(err error) {
	log.Fatal(err)
}

type csvCallback struct {
	carDir string
}

func (cc *csvCallback) OnSuccess(buf *Buffer, graphName, payloadCid, fsDetail string) {
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
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if isCreateAction {
		if _, err := f.Write([]byte("payload_cid,filename,detail")); err != nil {
			log.Fatal(err)
		}
	}

	if err := os.WriteFile(path.Join(cc.carDir, payloadCid+".car"), buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}

	if _, err := f.Write([]byte(fmt.Sprintf("%s,%s,%s", payloadCid, graphName, fsDetail))); err != nil {
		log.Fatal(err)
	}
}

func (cc *csvCallback) OnError(err error) {
	log.Fatal(err)
}

type errCallback struct{}

func (cc *errCallback) OnSuccess(*Buffer, string, string, string) {}
func (cc *errCallback) OnError(err error) {
	log.Fatal(err)
}

func CommPCallback(carDir string, rename, addPadding bool) GraphBuildCallback {
	return &commPCallback{carDir: carDir, rename: rename, addPadding: addPadding}
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
		return fmt.Errorf("slice size has been set as 0")
	}
	if parallel <= 0 {
		return fmt.Errorf("parallel has to be greater than 0")
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
			log.Infof("cumu-size: %d", cumuSize)
			log.Infof("%s", GenGraphName(graphName, graphSliceCount, sliceTotal))
			log.Infof("=================")
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
			log.Infof("first cut %d, seek start at %d, end at %d", firstCut, seekStart, seekEnd)
			log.Infof("----------------")
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
			log.Infof("cumu-size: %d", cumuSize+firstCut)
			log.Infof("%s", GenGraphName(graphName, graphSliceCount, sliceTotal))
			log.Infof("=================")
			cumuSize = 0
			graphFiles = make([]Finfo, 0)
			graphSliceCount++
			for seekEnd < fileSize-1 {
				seekStart = seekEnd + 1
				seekEnd = seekStart + sliceSize - 1
				if seekEnd >= fileSize-1 {
					seekEnd = fileSize - 1
				}
				log.Infof("following cut %d, seek start at %d, end at %d", seekEnd-seekStart+1, seekStart, seekEnd)
				log.Infof("----------------")
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
					log.Infof("cumu-size: %d", sliceSize)
					log.Infof("%s", GenGraphName(graphName, graphSliceCount, sliceTotal))
					log.Infof("=================")
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
		log.Infof("cumu-size: %d", cumuSize)
		log.Infof("%s", GenGraphName(graphName, graphSliceCount, sliceTotal))
		log.Infof("=================")
	}
	return nil
}
