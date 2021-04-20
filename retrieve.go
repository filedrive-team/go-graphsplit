package main

import (
	"context"
	"fmt"
	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dss "github.com/ipfs/go-datastore/sync"
	bstore "github.com/ipfs/go-ipfs-blockstore"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/go-merkledag"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipld/go-car"
	"golang.org/x/xerrors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func Import(path string, st car.Store) (cid.Cid, error) {
	f, err := os.Open(path)
	if err != nil {
		return cid.Undef, err
	}
	defer f.Close() //nolint:errcheck

	stat, err := f.Stat()
	if err != nil {
		return cid.Undef, err
	}

	file, err := files.NewReaderPathFile(path, f, stat)
	if err != nil {
		return cid.Undef, err
	}

	result, err := car.LoadCar(st, file)
	if err != nil {
		return cid.Undef, err
	}

	if len(result.Roots) != 1 {
		return cid.Undef, xerrors.New("cannot import car with more than one root")
	}

	return result.Roots[0], nil
}

func NodeWriteTo(nd files.Node, fpath string) error {
	switch nd := nd.(type) {
	case *files.Symlink:
		return os.Symlink(nd.Target, fpath)
	case files.File:
		f, err := os.Create(fpath)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(f, nd)
		if err != nil {
			return err
		}
		return nil
	case files.Directory:
		if !ExistDir(fpath) {
			err := os.Mkdir(fpath, 0777)
			if err != nil {
				return err
			}
		}

		entries := nd.Entries()
		for entries.Next() {
			child := filepath.Join(fpath, entries.Name())
			if err := NodeWriteTo(entries.Node(), child); err != nil {
				return err
			}
		}
		return entries.Err()
	default:
		return fmt.Errorf("file type %T at %q is not supported", nd, fpath)
	}
}

func ExistDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func CarTo(carPath, outputDir string, parallel int) {
	ctx := context.Background()
	bs2 := bstore.NewBlockstore(dss.MutexWrap(datastore.NewMapDatastore()))
	rdag := merkledag.NewDAGService(blockservice.New(bs2, offline.Exchange(bs2)))

	workerCh := make(chan func())
	go func() {
		defer close(workerCh)
		err := filepath.Walk(carPath, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			workerCh <- func() {
				log.Info(path)
				root, err := Import(path, bs2)
				if err != nil {
					log.Error("import error, ", err)
					return
				}
				nd, err := rdag.Get(ctx, root)
				if err != nil {
					log.Error("dagService.Get error, ", err)
					return
				}
				file, err := unixfile.NewUnixfsFile(ctx, rdag, nd)
				if err != nil {
					log.Error("NewUnixfsFile error, ", err)
					return
				}
				err = NodeWriteTo(file, outputDir)
				if err != nil {
					log.Error("NodeWriteTo error, ", err)
				}
			}
			return nil
		})
		if err != nil {
			log.Error("Walk path failed, ", err)
		}
	}()

	limitCh := make(chan struct{}, parallel)
	wg := sync.WaitGroup{}
	func() {
		for {
			select {
			case taskFunc, ok := <-workerCh:
				if !ok {
					return
				}
				limitCh <- struct{}{}
				wg.Add(1)
				go func() {
					defer func() {
						<-limitCh
						wg.Done()
					}()
					taskFunc()
				}()
			}
		}
	}()
	wg.Wait()
}

func Merge(dir string, parallel int) {
	wg := sync.WaitGroup{}
	limitCh := make(chan struct{}, parallel)
	mergeCh := make(chan string)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case fpath, ok := <-mergeCh:
				if !ok {
					return
				}
				limitCh <- struct{}{}
				wg.Add(1)
				go func() {
					defer func() {
						<-limitCh
						wg.Done()
					}()
					log.Info("merge to ", fpath)
					f, err := os.Create(fpath)
					if err != nil {
						log.Error("Create file failed, ", err)
						return
					}
					defer f.Close()
					for i := 0; ; i++ {
						chunkPath := fmt.Sprintf("%s.%08d", fpath, i)
						err := func(path string) error {
							chunkF, err := os.Open(path)
							if err != nil {
								if os.IsExist(err) {
									log.Error("Open file failed, ", err)
								}
								return err
							}
							defer chunkF.Close()
							_, err = io.Copy(f, chunkF)
							if err != nil {
								log.Error("io.Copy failed, ", err)
							}
							return err
						}(chunkPath)
						os.Remove(chunkPath)
						if err != nil {
							break
						}
					}
				}()
			}
		}
	}()
	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		matched, err := filepath.Match("*.00000000", fi.Name())
		if err != nil {
			log.Error("filepath.Match failed, ", err)
			return nil
		} else if matched {
			mergeCh <- strings.TrimSuffix(path, ".00000000")
		}
		return nil
	})
	if err != nil {
		log.Error("Walk path failed, ", err)
	}
	close(mergeCh)
	wg.Wait()
}
