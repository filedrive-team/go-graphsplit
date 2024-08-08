Go-graphsplit
==================
[![](https://img.shields.io/github/go-mod/go-version/filedrive-team/go-graphsplit)]()
[![](https://goreportcard.com/badge/github.com/filedrive-team/go-graphsplit)](https://goreportcard.com/report/github.com/filedrive-team/go-graphsplit)
[![](https://github.com/filedrive-team/go-graphsplit/actions/workflows/go.yml/badge.svg)]()
[![](https://img.shields.io/github/license/filedrive-team/go-graphsplit)](https://github.com/filedrive-team/go-graphsplit/blob/main/LICENSE)

> A tool for splitting a large dataset into graph slices to make deals in the Filecoin Network


When storing a large dataset, we need to split it into smaller pieces to fit the sector's size, which could generally be 32GiB or 64GiB.

If we make these data into a large tarball, chunk it into small pieces, and then make storage deals with miners with these pieces, on the side of storage, it will be pretty efficient and allow us to store hundreds of TiB data in a month. However, this way will also bring difficulties for data retrieval. Even if we only needed to retrieve a small file, we would first have to retrieve and download all the pieces of this tarball, decompress it, and find the specific file we needed.

Graphsplit can solve this problem. It takes advantage of IPLD protocol, follows the [Unixfs](https://github.com/ipfs/go-unixfs) format data structures, and regards the dataset or its sub-directory as a big graph, then cuts it into small graphs. Each small graph will keep its file system structure as possible as it used to be. After that, we only need to organize these small graphs into a car file. If one data piece has a complete file and we need to retrieve it, we only need to use payload CID to retrieve it through the lotus client, fetch it back, and get the file. Besides, Graphsplit will create a manifest.csv to save the mapping with graph slice name, payload CID, Piece CID, and the inner file structure.

Another advantage of Graphsplit is it can perfectly match IPFS. Like if you build an IPFS website as your Deal UI website, the inner file structure of each data piece can be shown on it, and it is easier for users to retrieve and download the data they stored.


## Build
```sh
git clone https://github.com/ipfs-force-community/go-graphsplit.git

cd go-graphsplit

make
```

## Usage

[See the work flow of graphsplit](doc/README.md)

Splitting dataset:
```sh
./graphsplit chunk \
# car-dir: folder for splitted smaller pieces, in form of .car
--car-dir=path/to/car-dir \
# slice-size: size for each pieces
--slice-size=17179869184 \
# parallel: number goroutines run when building ipld nodes
--parallel=2 \
# graph-name: it will use graph-name for prefix of smaller pieces
--graph-name=gs-test \
# calc-commp: calculation of pieceCID, default value is false. Be careful, a lot of cpu, memory and time would be consumed if slice size is very large.
--calc-commp=true \
# set true if want padding the car file to fit piece size
--add-padding=false \
# set true if want using piececid to name the chunk file
--rename=true \
/path/to/dataset
```

> Notes: A manifest.csv will created to save the mapping with graph slice name, the payload cid and slice inner structure. As following:

```sh
cat /path/to/car-dir/manifest.csv
payload_cid,filename,detail
ba...,graph-slice-name.car,inner-structure-json
```

If set `--calc-commp=true`, two another fields would be add to manifest.csv

```sh
cat /path/to/car-dir/manifest.csv
payload_cid,filename,piece_cid,piece_size,detail
ba...,graph-slice-name.car,baga...,16646144,inner-structure-json
```

Import car file to IPFS: 
```sh
ipfs dag import /path/to/car-dir/car-file
```

Restore files:
```sh
# car-path: directory or file, in form of .car
# output-dir: usually just be the same as /path/to/output-dir
# parallel: number goroutines run when restoring
./graphsplit restore \
--car-path=/path/to/car-path \
--output-dir=/path/to/output-dir \
--parallel=2
```

PieceCID Calculation for a single car file:


```shell
# Calculate pieceCID for a single car file
# 
./graphsplit commP /path/to/carfile
```

## Contribute

PRs are welcome!

## License

MIT
