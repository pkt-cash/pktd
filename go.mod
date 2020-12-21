module github.com/pkt-cash/pktd

go 1.14

replace (
	git.schwanenlied.me/yawning/bsaes.git => github.com/Yawning/bsaes v0.0.0-20180720073208-c0276d75487e
	github.com/coreos/bbolt v1.3.5 => go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	google.golang.org/grpc v1.29.1 => google.golang.org/grpc v1.29.0
)

require (
	filippo.io/edwards25519 v1.0.0-beta.2.0.20201218140448-c5477978affe // indirect
	git.schwanenlied.me/yawning/bsaes.git v0.0.0-20190320102049-26d1add596b6 // indirect
	github.com/NebulousLabs/go-upnp v0.0.0-20181203152547-b32978b8ccbf
	github.com/Yawning/aez v0.0.0-20180408160647-ec7426b44926
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/arl/statsviz v0.2.3-0.20201213204859-dafc5bde1f65
	github.com/btcsuite/winsvc v1.0.0
	github.com/coreos/bbolt v1.3.5 // indirect
	github.com/coreos/etcd v3.3.22+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/blake2b v1.0.0
	github.com/dustin/go-humanize v1.0.1-0.20200219035652-afde56e7acac // indirect
	github.com/emirpasic/gods v1.12.1-0.20201118132343-79df803e554c
	github.com/frankban/quicktest v1.11.3 // indirect
	github.com/fsnotify/fsnotify v1.4.10-0.20200417215612-7f4cf4dd2b52 // indirect
	github.com/go-errors/errors v1.1.1
	github.com/go-openapi/errors v0.19.9 // indirect
	github.com/go-openapi/strfmt v0.19.11 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/golang/snappy v0.0.3-0.20201103224600-674baa8c7fc3
	github.com/google/uuid v1.1.2 // indirect
	github.com/gorilla/websocket v1.4.3-0.20200912193213-c3dd95aea977
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.1-0.20190118093823-f849b5445de4
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/hdevalence/ed25519consensus v0.0.0-20201207055737-7fde80a9d5ff
	github.com/jackpal/gateway v1.0.7-0.20201119002851-2ba2a7cd5c7b
	github.com/jackpal/go-nat-pmp v1.0.2
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/johnsonjh/goc25519sm v1.4.5-0.20201217171032-0b745b266201
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/json-iterator/go v1.1.11-0.20201118013158-e6b9536d3649
	github.com/juju/testing v0.0.0-20201216035041-2be42bba85f3 // indirect
	github.com/kkdai/bstream v1.0.0
	github.com/lightninglabs/protobuf-hex-display v1.3.3-0.20191212020323-b444784ce75d
	github.com/ltcsuite/ltcd v0.20.1-beta.0.20201210074626-c807bfe31ef0
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/miekg/dns v1.1.36-0.20201218191609-23c4faca9d32
	github.com/minio/sha256-simd v0.1.2-0.20190917233721-f675151bb5e1
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/modern-go/reflect2 v1.0.2-0.20200602030031-7e6ae53ffa0b // indirect
	github.com/nxadm/tail v1.4.6-0.20201001195649-edf6bc2dfc36 // indirect
	github.com/onsi/ginkgo v1.14.3-0.20201215232527-efb9e6987c00
	github.com/onsi/gomega v1.10.5-0.20201208201658-3ed17884e444
	github.com/pkg/errors v0.9.2-0.20201214064552-5dd12d0cfe7f // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	github.com/soheilhy/cmux v0.1.5-0.20181025144106-8a8ea3c53959 // indirect
	github.com/sony/sonyflake v1.0.1-0.20200827011719-848d664ceea4
	github.com/stretchr/testify v1.6.2-0.20201103103935-92707c0b2d50
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/tv42/zbase32 v0.0.0-20190604154422-aacc64a8f915
	github.com/urfave/cli v1.22.2-0.20191024042601-850de854cda0
	gitlab.com/NebulousLabs/fastrand v0.0.0-20181126182046-603482d69e40 // indirect
	gitlab.com/NebulousLabs/go-upnp v0.0.0-20181011194642-3a71999ed0d3 // indirect
	go.etcd.io/bbolt v1.3.5
	go.mongodb.org/mongo-driver v1.4.4 // indirect
	go.uber.org/goleak v1.1.11-0.20200902203756-89d54f0adef2
	go.uber.org/multierr v1.6.1-0.20201124182017-e015acf18bb3 // indirect
	go.uber.org/zap v1.16.1-0.20201211181745-a68efdbdd15b // indirect
	go4.org v0.0.0-20201209231011-d4a079459e60
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620
	golang.org/x/net v0.0.0-20201216054612-986b41b23924
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20201218084310-7d0127a74742
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf
	golang.org/x/text v0.3.5-0.20201208001344-75a595aef632 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324
	golang.org/x/tools v0.0.0-20201218024724-ae774e9781d2 // indirect
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d // indirect
	google.golang.org/grpc v1.33.1
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/macaroon-bakery.v2 v2.0.1
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
	honnef.co/go/tools v0.2.0-0.dev.0.20201215111046-903cdb5ed2f9 // indirect
	sigs.k8s.io/yaml v1.2.1-0.20201021160022-8aabd9a1b2a7 // indirect
)
