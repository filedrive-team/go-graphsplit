package graphsplit

import (
	"context"
	"fmt"
	ipld "github.com/ipfs/go-ipld-format"
	"testing"
)

type errCallbackTest2 struct {
	rootCID string
}

func (cc *errCallbackTest2) OnSuccess(node ipld.Node, str string) {
	cc.rootCID = fmt.Sprintf("%s", node.Cid())
}
func (cc *errCallbackTest2) OnError(err error) {
	log.Fatal(err)
}
func TestChunk(t *testing.T) {
	ctx := context.Background()
	sliceSize := 2000
	parentPath := ""
	targetPath := "./example"
	carDir := "./chunkDir"
	graphName := "abc"
	parallel := 2
	var ecb errCallbackTest2
	err := Chunk(ctx, int64(sliceSize), parentPath, targetPath, carDir, graphName, parallel, &ecb)
	if err != nil {
		t.Error("chunk过程发生错误", err)
	}
}
