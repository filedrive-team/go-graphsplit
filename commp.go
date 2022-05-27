package graphsplit

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/filecoin-project/go-commp-utils/ffiwrapper"
	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
	"golang.org/x/xerrors"
)

type CommPRet struct {
	Root cid.Cid
	PayloadSize int64
	Size abi.UnpaddedPieceSize
}

// almost copy paste from https://github.com/filecoin-project/lotus/node/impl/client/client.go#L749-L770
func CalcCommP(ctx context.Context, inpath string, rename bool) (*CommPRet, error) {
	dir, _ := path.Split(inpath)
	// Hard-code the sector type to 32GiBV1_1, because:
	// - ffiwrapper.GeneratePieceCIDFromFile requires a RegisteredSealProof
	// - commP itself is sector-size independent, with rather low probability of that changing
	//   ( note how the final rust call is identical for every RegSP type )
	//   https://github.com/filecoin-project/rust-filecoin-proofs-api/blob/v5.0.0/src/seal.rs#L1040-L1050
	//
	// IF/WHEN this changes in the future we will have to be able to calculate
	// "old style" commP, and thus will need to introduce a version switch or similar
	arbitraryProofType := abi.RegisteredSealProof_StackedDrg32GiBV1_1

	st, err := os.Stat(inpath)
	if err != nil  {
		return nil, err
	}

	if st.IsDir() {
		return nil, fmt.Errorf("path %s is dir", inpath)
	}
	payloadSize := st.Size()

	rdr, err := os.Open(inpath)
	if err != nil {
		return nil, err
	}
	defer rdr.Close() //nolint:errcheck

	stat, err := rdr.Stat()
	if err != nil {
		return nil, err
	}

	// check that the data is a car file; if it's not, retrieval won't work
	_, err = car.ReadHeader(bufio.NewReader(rdr))
	if err != nil {
		return nil, xerrors.Errorf("not a car file: %w", err)
	}

	if _, err := rdr.Seek(0, io.SeekStart); err != nil {
		return nil, xerrors.Errorf("seek to start: %w", err)
	}

	pieceReader, pieceSize := padreader.New(rdr, uint64(stat.Size()))
	commP, err := ffiwrapper.GeneratePieceCIDFromFile(arbitraryProofType, pieceReader, pieceSize)
	if err != nil {
		return nil, xerrors.Errorf("computing commP failed: %w", err)
	}

	if padreader.PaddedSize(uint64(payloadSize)) != pieceSize {
		return nil, xerrors.Errorf("assert car(%s) file to piece fail payload size(%d) piece size (%d)", inpath, payloadSize, pieceSize)
	}
	if rename {
		piecePath := path.Join(dir, commP.String())
		err = os.Rename(inpath, piecePath)
		if err != nil {
			return nil, xerrors.Errorf("rename car(%s) file to piece %w", inpath, err)
		}
	}
	return &CommPRet{
		Root: commP,
		Size: pieceSize,
		PayloadSize: payloadSize,
	}, nil
}

