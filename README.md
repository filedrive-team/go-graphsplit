go-graphsplit
==================
> A tool for splitting large dataset into graph slices feat for making deal in filecoin network.


When storing large dataset into filecoin network we have to split it into smaller pieces to feat for storage miner's sector size which could be 32GiB or 64GiB.

At first, we make a large tar ball from the dataset and chunking the tar ball into pieces then make deal with storage miner for each piece. We did this way for a while until we realizing it brings a retrieval difficulty. Even if we only need to retrieve a small file in the dataset, we had to retrieve all the pieces of the tar ball at first. 

Graphsplit try to solve the problem we facing above. It takes advantage of IPLD concepts, following the [unixfs](https://github.com/ipfs/go-unixfs) format datastructures. It regards dataset or it's sub-directory as a big graph and then cut big graph into small graphs. The small graphs will keep its file system structure as possible as its in big graph. Finally we organize small graph into a car file according to [unixfs](https://github.com/ipfs/go-unixfs).

## Usage


## Contribute

PRs are welcome!


## License

MIT

