package graphsplit

import (
	"context"
	"fmt"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-datastore"
	dss "github.com/ipfs/go-datastore/sync"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	unixfile "github.com/ipfs/go-unixfs/file"
	"testing"
)

type errCallbackTest struct {
	rootCID string
}

func (cc *errCallbackTest) OnSuccess(node ipld.Node, str string) {
	cc.rootCID = fmt.Sprintf("%s", node.Cid())
}
func (cc *errCallbackTest) OnError(err error) {
	log.Fatal(err)
}
func TestImport(t *testing.T) {
	ctx := context.Background()
	tarPath := "./example"
	carDir := "./chunkDir"
	var ecb errCallbackTest
	err := Chunk(ctx, 2000, "", tarPath, carDir, "abc", 2, &ecb)
	if err != nil {
		t.Error("TestImport时chunk发生错误", err)
	}
	bs2 := bstore.NewBlockstore(dss.MutexWrap(datastore.NewMapDatastore()))
	cid, err := Import("./chunkDir/"+fmt.Sprintf("%s.car", ecb.rootCID), bs2)
	if err != nil {
		t.Error("import发生错误", err)
	}
	if cid.String() != fmt.Sprintf("%s", ecb.rootCID) {
		t.Error("chunk和import的CID不一致", err)
	}
}

func TestNodeWriteTo(t *testing.T) {
	ctx := context.Background()
	bs2 := bstore.NewBlockstore(dss.MutexWrap(datastore.NewMapDatastore()))
	rdag := merkledag.NewDAGService(blockservice.New(bs2, offline.Exchange(bs2)))
	var ecb errCallbackTest
	tarPath := "./example"
	carDir := "./chunkDir"
	err := Chunk(ctx, 2000, "", tarPath, carDir, "abc", 2, &ecb)
	if err != nil {
		t.Error("TestImport时chunk发生错误", err)
	}
	root, err := Import("./chunkDir/"+fmt.Sprintf("%s.car", ecb.rootCID), bs2)
	if err != nil {
		t.Error("TestNodeWriteTo时import发生错误", err)
	}
	nd, err := rdag.Get(ctx, root)
	file, err := unixfile.NewUnixfsFile(ctx, rdag, nd)
	if err != nil {
		t.Error("产生新的unixfsFile时产生错误", err)
	}
	outputDir := "./RestoreDir"
	err = NodeWriteTo(file, outputDir)
	if err != nil {
		t.Error("TestNodeWriteTo报错", err)
	}
}

/*func TestCarTo(t *testing.T) {
	carPath := "./chunkDir"
	outputDir := "./RestoreDir"
	parallel := 2
	CarTo(carPath, outputDir, parallel)
}

func TestMerge(t *testing.T) {
	dir :=
	parallel := 2
	Merge(dir, parallel)
}
*/
