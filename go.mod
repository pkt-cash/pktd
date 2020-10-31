module github.com/pkt-cash/pktd

go 1.14

replace github.com/pkt-cash/pktd/goleveldb => ./goleveldb

require (
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/btcsuite/winsvc v1.0.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/blake2b v1.0.0
	github.com/decred/go-socks v1.1.0
	github.com/emirpasic/gods v1.12.1-0.20200630092735-7e2349589531
	github.com/fsnotify/fsnotify v1.4.10-0.20200417215612-7f4cf4dd2b52 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/websocket v1.4.3-0.20200912193213-c3dd95aea977
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/json-iterator/go v1.1.10
	github.com/kkdai/bstream v1.0.0
	github.com/lightningnetwork/lnd/queue v1.0.5-0.20201016111222-d12f76fd6d48
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pkt-cash/pktd/goleveldb v0.0.0
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/ulikunitz/xz v0.5.9-0.20200819203118-cedfac7102e1 // indirect
	go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	google.golang.org/genproto v0.0.0-20201021134325-0d71844de594 // indirect
	google.golang.org/grpc v1.34.0-dev.0.20201021230544-4e8458e5c638
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)
