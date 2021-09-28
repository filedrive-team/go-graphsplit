package dataset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	dsrpc "github.com/beeleelee/go-ds-rpc"
	dsmongo "github.com/beeleelee/go-ds-rpc/ds-mongo"
	"github.com/filedrive-team/go-ds-cluster/clusterclient"
	clustercfg "github.com/filedrive-team/go-ds-cluster/config"
	"github.com/filedrive-team/go-graphsplit"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dsmount "github.com/ipfs/go-datastore/mount"
	dss "github.com/ipfs/go-datastore/sync"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	ipld "github.com/ipfs/go-ipld-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
)

var log = logging.Logger("graphsplit/dataset")

func Import(ctx context.Context, target, mongouri, dsclusterCfg string) error {
	recordPath := path.Join(target, record_json)
	// check if record.json has data
	records, err := readRecords(recordPath)
	if err != nil {
		return err
	}
	var ds ds.Datastore

	if dsclusterCfg != "" {
		cfg, err := clustercfg.ReadConfig(dsclusterCfg)
		if err != nil {
			return err
		}
		ds, err = clusterclient.NewClusterClient(context.Background(), cfg)
		if err != nil {
			return err
		}
	} else {
		// go-ds-rpc dsmongo
		client, err := dsmongo.NewMongoStoreClient(mongouri)
		if err != nil {
			return err
		}
		ds, err = dsrpc.NewDataStore(client)
		if err != nil {
			return err
		}
	}

	ds = dsmount.New([]dsmount.Mount{
		{
			Prefix:    bstore.BlockPrefix,
			Datastore: ds,
		},
	})

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

		fileNode, err := buildFileNodeRetry(5, item, dagServ, cidBuilder)
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
	}
	err = saveRecords(records, recordPath)
	if err != nil {
		ferr = err
	}
	fmt.Printf("total %d files, imported %d files, %.2f %%\n", totol_files, len(records), float64(len(records))/float64(totol_files)*100)
	return ferr
}

func buildFileNodeRetry(times int, item graphsplit.Finfo, dagServ ipld.DAGService, cidBuilder cid.Builder) (root ipld.Node, err error) {
	for i := 0; i < times; i++ {
		log.Infof("import file: %s, try times: %d", item.Path, i)
		if root, err = graphsplit.BuildFileNode(item, dagServ, cidBuilder); err == nil {
			return root, nil
		}
		// should wait a second if io.EOF
		if err == io.EOF {
			log.Infof("io.EOF wait %d ms", 800)
			time.Sleep(time.Millisecond * 800)
		}
	}
	return
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
	return ioutil.WriteFile(path, bs, 0666)
}
