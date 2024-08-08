package graphsplit

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/filecoin-project/go-commp-utils/v2"
	"github.com/filecoin-project/go-padreader"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
)

type CommPRet struct {
	Root        cid.Cid
	PayloadSize int64
	Size        abi.UnpaddedPieceSize
}

// almost copy paste from https://github.com/filecoin-project/lotus/node/impl/client/client.go#L749-L770
func CalcCommP(ctx context.Context, inpath string, rename, addPadding bool) (*CommPRet, error) {
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
	if err != nil {
		return nil, err
	}

	if st.IsDir() {
		return nil, fmt.Errorf("path %s is dir", inpath)
	}
	payloadSize := st.Size()

	rdr, err := os.OpenFile(inpath, os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	defer rdr.Close() //nolint:errcheck

	stat, err := rdr.Stat()
	if err != nil {
		return nil, err
	}
	carSize := stat.Size()
	// check that the data is a car file; if it's not, retrieval won't work
	_, err = car.ReadHeader(bufio.NewReader(rdr))
	if err != nil {
		return nil, fmt.Errorf("not a car file: %w", err)
	}

	if _, err := rdr.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	pieceReader, pieceSize := padreader.New(rdr, uint64(carSize))
	commP, err := commp.GeneratePieceCIDFromFile(arbitraryProofType, pieceReader, pieceSize)
	if err != nil {
		return nil, fmt.Errorf("computing commP failed: %w", err)
	}

	if padreader.PaddedSize(uint64(payloadSize)) != pieceSize {
		return nil, fmt.Errorf("assert car(%s) file to piece fail payload size(%d) piece size (%d)", inpath, payloadSize, pieceSize)
	}
	if addPadding {
		// make sure fd point to the end of file
		// better to check within carv1.PadCar, for now is a workaround
		if _, err := rdr.Seek(carSize, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek to start: %w", err)
		}
		if err := PadCar(rdr, carSize); err != nil {
			return nil, fmt.Errorf("failed to pad car file: %w", err)
		}
	}
	if rename {
		piecePath := path.Join(dir, commP.String())
		err = os.Rename(inpath, piecePath)
		if err != nil {
			return nil, fmt.Errorf("rename car(%s) file to piece %w", inpath, err)
		}
	}
	return &CommPRet{
		Root:        commP,
		Size:        pieceSize,
		PayloadSize: payloadSize,
	}, nil
}

func CalcCommPV2(buf *Buffer, addPadding bool) (*CommPRet, error) {
	arbitraryProofType := abi.RegisteredSealProof_StackedDrg32GiBV1_1

	// check that the data is a car file; if it's not, retrieval won't work
	_, err := car.ReadHeader(bufio.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("not a car file: %w", err)
	}
	buf.SeekStart()

	carSize := int64(buf.Len())
	pieceReader, pieceSize := padreader.New(buf, uint64(carSize))
	commP, err := commp.GeneratePieceCIDFromFile(arbitraryProofType, pieceReader, pieceSize)
	if err != nil {
		return nil, fmt.Errorf("computing commP failed: %w", err)
	}

	if padreader.PaddedSize(uint64(carSize)) != pieceSize {
		return nil, fmt.Errorf("assert file to piece fail payload size(%d) piece size (%d)", carSize, pieceSize)
	}

	if addPadding {
		// make sure fd point to the end of file
		// better to check within carv1.PadCar, for now is a workaround
		buf.Seek(int(carSize))
		if err := PadCar(buf, carSize); err != nil {
			return nil, fmt.Errorf("failed to pad car file: %w", err)
		}
	}

	return &CommPRet{
		Root:        commP,
		Size:        pieceSize,
		PayloadSize: int64(carSize),
	}, nil
}
