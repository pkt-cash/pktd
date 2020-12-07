module github.com/pkt-cash/pktd

go 1.14

replace github.com/pkt-cash/pktd/goleveldb => ./goleveldb

require (
	filippo.io/edwards25519 v1.0.0-beta.1 // indirect
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/arl/statsviz v0.2.2-0.20201115121518-5ea9f0cf1bd1
	github.com/btcsuite/winsvc v1.0.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/emirpasic/gods v1.12.1-0.20200630092735-7e2349589531
	github.com/fsnotify/fsnotify v1.4.10-0.20200417215612-7f4cf4dd2b52 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/websocket v1.4.3-0.20200912193213-c3dd95aea977
	github.com/hdevalence/ed25519consensus v0.0.0-20201207055737-7fde80a9d5ff
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/johnsonjh/goc25519sm v1.3.3-0.20201206213152-3c8fba1589f3
	github.com/json-iterator/go v1.1.11-0.20200806011408-6821bec9fa5c
	github.com/kkdai/bstream v1.0.0
	github.com/lightningnetwork/lnd/queue v1.0.5-0.20201016111222-d12f76fd6d48
	github.com/minio/sha256-simd v0.1.2-0.20190917233721-f675151bb5e1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/pkt-cash/pktd/goleveldb v0.0.0
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	go.uber.org/goleak v1.1.11-0.20200902203756-89d54f0adef2
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb // indirect
	golang.org/x/sys v0.0.0-20201204225414-ed752295db88 // indirect
	golang.org/x/tools v0.0.0-20201206230334-368bee879bfd // indirect
	google.golang.org/genproto v0.0.0-20201204160425-06b3db808446 // indirect
	google.golang.org/grpc v1.34.0
)
