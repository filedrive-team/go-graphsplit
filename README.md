Go-graphsplit
==================
> A tool for splitting large dataset into graph slices fit for making deal in the Filecoin Network.


When storing a large dataset in the Filecoin Network, we have to split it into smaller pieces to fit for the size of sector, which could be 32GiB or 64GiB.

At first, we made the dataset into a large tar ball, did chunking this tar ball into small pieces, and then make deals with storage miners with these pieces. We did this way for a while until we realized that it brought a difficulty for data retrieval. Even if we only needed to retrieve a small file in the dataset, we had to retrieve all the pieces of the tar ball at first. 

Graphsplit has solved the problem we faced above. It takes advantage of IPLD concepts, following the [unixfs](https://github.com/ipfs/go-unixfs) format datastructures. It regards the dataset or it's sub-directory as a big graph and then cut it into small graphs. Each small graph will keep its file system structure as possible as its used be. After that, we only need to organize these small graphs into a car file according to [unixfs](https://github.com/ipfs/go-unixfs).

## Build
```sh
git clone https://github.com/filedrive-team/go-graphsplit.git
cd go-graphsplit
make
```

## Usage

Splitting dataset:
```sh
# car-dir: folder for splitted smaller pieces, in form of .car
# slice-size: size for each pieces
# parallel: number goroutines run when building ipld nodes
# graph-name: it will use graph-name for prefix of smaller pieces
# parent-path: usually just be the same as /path/to/dataset, it's just a method to figure out relative path when building IPLD graph
./graphsplit chunk \
--car-dir=path/to/car-dir \
--slice-size=17179869184 \
--parallel=2 \
--graph-name=gs-test \
--parent-path=/path/to/dataset \
/path/to/dataset
```
Notes: A manifest.csv will created to save the mapping with graph slice name and the payload cid. As following:
```sh
cat /path/to/car-dir/manifest.csv
payload_cid,filename
Qm...,graph-slice-name.car
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

## Contribute

PRs are welcome!


## License

MIT

