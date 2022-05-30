package graphsplit

import (
	"bufio"
	"context"
	"fmt"
	"github.com/filecoin-project/go-commp-utils/writer"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
	"golang.org/x/xerrors"
	"io"
	"os"
	"path"
)

type CommPRet struct {
	Root        cid.Cid
	PayloadSize int64
	Size        abi.UnpaddedPieceSize
}

// almost copy paste from https://github.com/filecoin-project/lotus/node/impl/client/client.go#L749-L770
func CalcCommP(ctx context.Context, inpath string, rename bool) (*CommPRet, error) {
	dir, _ := path.Split(inpath)

	rdr, err := os.Open(inpath)
	if err != nil {
		return nil, err
	}
	defer rdr.Close() //nolint:errcheck

	st, err := os.Stat(inpath)
	if err != nil {
		return nil, err
	}

	if st.IsDir() {
		return nil, fmt.Errorf("path %s is dir", inpath)
	}

	payloadSize := st.Size()

	// check that the data is a car file; if it's not, retrieval won't work
	_, err = car.ReadHeader(bufio.NewReader(rdr))
	if err != nil {
		return nil, fmt.Errorf("not a car file: %w", err)
	}

	if _, err := rdr.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	w := &writer.Writer{}
	_, err = io.CopyBuffer(w, rdr, make([]byte, writer.CommPBuf))
	if err != nil {
		return nil, fmt.Errorf("copy into commp writer: %w", err)
	}

	commP, err := w.Sum()
	if err != nil {
		return nil, fmt.Errorf("computing commP failed: %w", err)
	}

	if rename {
		piecePath := path.Join(dir, commP.PieceCID.String())
		err = os.Rename(inpath, piecePath)
		if err != nil {
			return nil, xerrors.Errorf("rename car(%s) file to piece %w", inpath, err)
		}
	}

	return &CommPRet{
		Root:        commP.PieceCID,
		Size:        commP.PieceSize.Unpadded(),
		PayloadSize: payloadSize,
	}, nil
}
