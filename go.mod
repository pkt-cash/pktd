module github.com/pkt-cash/pktd

go 1.14

replace github.com/pkt-cash/pktd/goleveldb v1.0.2 => ./goleveldb

require (
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/btcsuite/winsvc v1.0.0
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/blake2b v1.0.0
	github.com/decred/go-socks v1.1.0
	github.com/emirpasic/gods v1.12.1-0.20200630092735-7e2349589531
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/gorilla/websocket v1.4.3-0.20200912193213-c3dd95aea977
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/json-iterator/go v1.1.11-0.20200806011408-6821bec9fa5c
	github.com/kkdai/bstream v1.0.0
	github.com/kr/text v0.2.0 // indirect
	github.com/lightningnetwork/lnd/queue v1.0.5-0.20201016111222-d12f76fd6d48
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2-0.20200602030031-7e6ae53ffa0b // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pkt-cash/pktd/goleveldb v1.0.2
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	github.com/stretchr/testify v1.6.2-0.20200818115829-54d05a4e1844 // indirect
	go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	google.golang.org/genproto v0.0.0-20201021134325-0d71844de594 // indirect
	google.golang.org/grpc v1.34.0-dev.0.20201021230544-4e8458e5c638
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)
