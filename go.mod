module github.com/pkt-cash/pktd

go 1.14

require (
	github.com/LK4D4/trylock v0.0.0-20191027065348-ff7e133a5c54
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/btcsuite/winsvc v1.0.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/blake2b v1.0.0
	github.com/decred/go-socks v1.1.0
	github.com/emirpasic/gods v1.12.1-0.20200630092735-7e2349589531
	github.com/gorilla/websocket v1.4.3-0.20200822210332-78ab81e2420a
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/kkdai/bstream v1.0.0
	github.com/lightningnetwork/lnd/queue v1.0.5-0.20200828124145-e4764a67cc41
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	google.golang.org/genproto v0.0.0-20200829155447-2bf3329a0021 // indirect
	google.golang.org/grpc v1.33.0-dev.0.20200828165940-d8ef479ab79a
	google.golang.org/protobuf v1.25.0 // indirect
)

replace github.com/pkt-cash/pktd/goleveldb => ./goleveldb
