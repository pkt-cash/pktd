module github.com/pkt-cash/pktd

go 1.14

require (
	github.com/LK4D4/trylock v0.0.0-20191027065348-ff7e133a5c54
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd
	github.com/btcsuite/golangcrypto v0.0.0-20150304025918-53f62d9b43e8
	github.com/pkt-cash/pktd/goleveldb v1.0.0
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792
	github.com/btcsuite/winsvc v1.0.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/blake2b v1.0.0
	github.com/emirpasic/gods v1.12.1-0.20200630092735-7e2349589531
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/kkdai/bstream v1.0.0
	github.com/lightningnetwork/lnd/queue v1.0.5-0.20200828124145-e4764a67cc41
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200828194041-157a740278f4 // indirect
	golang.org/x/text v0.3.4-0.20200826142016-a8b467125457 // indirect
	google.golang.org/genproto v0.0.0-20200829155447-2bf3329a0021 // indirect
	google.golang.org/grpc v1.33.0-dev.0.20200828165940-d8ef479ab79a
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)

replace github.com/pkt-cash/pktd/goleveldb => ./goleveldb
