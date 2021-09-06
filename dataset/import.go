package dataset

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path"
)

func Import(ctx context.Context, target string) error {
	// check if record.json has data
	records, err := readRecords(path.Join(target, record_json))
	if err != nil {
		return err
	}
	// cidbuilder

	// read files

	_ = records
	return nil
}

func readRecords(path string) (map[string]*MetaData, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rearray := make([]*MetaData, 0)
	err = json.Unmarshal(bs, &rearray)
	if err != nil {
		return nil, err
	}
	res := make(map[string]*MetaData)
	for _, md := range rearray {
		res[md.Path] = md
	}
	return res, nil
}
