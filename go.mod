module github.com/filedrive-team/go-graphsplit

go 1.15

require (
	github.com/beeleelee/go-ds-rpc v0.1.0 // this needs to be updated too https://github.com/beeleelee/go-ds-rpc/pull/3
	github.com/filecoin-project/go-commp-utils v0.1.3
	github.com/filecoin-project/go-padreader v0.0.1 // indirect
	github.com/filecoin-project/go-state-types v0.1.4
	github.com/ipfs/go-blockservice v0.2.1
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.5.1
	github.com/ipfs/go-ipfs-blockstore v1.1.2
	github.com/ipfs/go-ipfs-chunker v0.0.5
	github.com/ipfs/go-ipfs-exchange-offline v0.1.1
	github.com/ipfs/go-ipfs-files v0.0.9
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/ipfs/go-merkledag v0.5.1
	github.com/ipfs/go-unixfs v0.3.1
	github.com/ipld/go-car v0.3.3
	github.com/ipld/go-ipld-prime v0.16.0
	github.com/urfave/cli/v2 v2.6.0
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/xerrors v0.0.0-20220517211312-f3a8303e98df
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
