package dataset

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	dsrpc "github.com/beeleelee/go-ds-rpc"
	dsmongo "github.com/beeleelee/go-ds-rpc/ds-mongo"
	"github.com/filedrive-team/go-graphsplit"
	"github.com/ipfs/go-blockservice"
	dss "github.com/ipfs/go-datastore/sync"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
)

var log = logging.Logger("graphsplit/dataset")

func Import(ctx context.Context, target, mongouri string) error {
	recordPath := path.Join(target, record_json)
	// check if record.json has data
	records, err := readRecords(recordPath)
	if err != nil {
		return err
	}

	// go-ds-rpc dsmongo
	client, err := dsmongo.NewMongoStoreClient(mongouri)
	if err != nil {
		return err
	}
	ds, err := dsrpc.NewDataStore(client)
	if err != nil {
		return err
	}

	bs2 := bstore.NewBlockstore(dss.MutexWrap(ds))
	dagServ := merkledag.NewDAGService(blockservice.New(bs2, offline.Exchange(bs2)))

	// cidbuilder
	cidBuilder, err := merkledag.PrefixForCidVersion(0)
	if err != nil {
		return err
	}

	// read files
	allfiles, err := graphsplit.GetFileList([]string{target})
	if err != nil {
		return err
	}
	totol_files := len(allfiles)
	var ferr error
	files := graphsplit.GetFileListAsync([]string{target})
	for item := range files {
		// ignore record_json
		if item.Name == record_json {
			totol_files -= 1
			continue
		}

		// ignore file which has been imported
		if _, ok := records[item.Path]; ok {
			continue
		}
		log.Infof("import file: %s", item.Path)
		fileNode, err := graphsplit.BuildFileNode(item, dagServ, cidBuilder)
		if err != nil {
			ferr = err
			break
		}
		records[item.Path] = &MetaData{
			Path: item.Path,
			Name: item.Name,
			Size: item.Info.Size(),
			CID:  fileNode.Cid().String(),
		}
		err = saveRecords(records, recordPath)
		if err != nil {
			ferr = err
			break
		}
	}
	fmt.Printf("total %d files, imported %d files, %.2f %%\n", len(allfiles), len(records), float64(len(records))/float64(totol_files)*100)
	return ferr
}

func readRecords(path string) (map[string]*MetaData, error) {
	res := make(map[string]*MetaData)
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return nil, err
	}

	err = json.Unmarshal(bs, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func saveRecords(records map[string]*MetaData, path string) error {
	bs, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, bs, 0o666)
}
