module github.com/pkt-cash/pktd

go 1.14

replace (
	git.schwanenlied.me/yawning/bsaes.git => github.com/Yawning/bsaes v0.0.0-20180720073208-c0276d75487e
	go.etcd.io/bbolt v1.3.5 => go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	google.golang.org/grpc v1.29.1 => google.golang.org/grpc v1.34.0
	google.golang.org/grpc/naming@latest => google.golang.org/grpc v1.29.1
)

require (
	git.schwanenlied.me/yawning/bsaes.git v0.0.0-20180720073208-c0276d75487e // indirect
	github.com/NebulousLabs/fastrand v0.0.0-20181203155948-6fb6489aac4e // indirect
	github.com/NebulousLabs/go-upnp v0.0.0-20180202185039-29b680b06c82
	github.com/Yawning/aez v0.0.0-20180114000226-4dad034d9db2
	github.com/aead/chacha20 v0.0.0-20180709150244-8b13a72661da
	github.com/aead/siphash v1.0.1
	github.com/arl/statsviz v0.2.2-0.20201124191121-aeacce1395b7
	github.com/btcsuite/winsvc v1.0.0
	github.com/coreos/bbolt v1.3.6-0.20200807205753-f6be82302843 // indirect
	github.com/coreos/etcd v3.3.22+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/blake2b v1.0.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/emirpasic/gods v1.12.1-0.20200630092735-7e2349589531
	github.com/fsnotify/fsnotify v1.4.10-0.20200417215612-7f4cf4dd2b52 // indirect
	github.com/go-errors/errors v1.0.1
	github.com/go-openapi/strfmt v0.19.5 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.4.3
	github.com/golang/snappy v0.0.2
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/gorilla/websocket v1.4.3-0.20200912193213-c3dd95aea977
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway v1.14.3
	github.com/jackpal/gateway v1.0.5
	github.com/jackpal/go-nat-pmp v0.0.0-20170405195558-28a68d0c24ad
	github.com/jedib0t/go-pretty v4.3.0+incompatible
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/json-iterator/go v1.1.11-0.20200806011408-6821bec9fa5c
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20190806202954-0232dcc7464d // indirect
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/retry v0.0.0-20180821225755-9058e192b216 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/juju/utils v0.0.0-20180820210520-bf9cc5bdd62d // indirect
	github.com/juju/version v0.0.0-20180108022336-b64dbd566305 // indirect
	github.com/kkdai/bstream v1.0.0
	github.com/lightninglabs/protobuf-hex-display v1.3.3-0.20191212020323-b444784ce75d
	github.com/ltcsuite/ltcd v0.0.0-20190101042124-f37f8bf35796
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/miekg/dns v0.0.0-20171125082028-79bfde677fa8
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nxadm/tail v1.4.6-0.20201001195649-edf6bc2dfc36 // indirect
	github.com/onsi/ginkgo v1.14.3-0.20201013214636-dfe369837f25
	github.com/onsi/gomega v1.10.3
	github.com/prometheus/client_golang v0.9.3
	github.com/sethgrid/pester v1.1.1-0.20200617174401-d2ad9ec9a8b6
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/tv42/zbase32 v0.0.0-20160707012821-501572607d02
	github.com/urfave/cli v1.18.0
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.etcd.io/bbolt v1.3.5
	go.uber.org/zap v1.14.1 // indirect
	github.com/sony/sonyflake v1.0.1-0.20200827011719-848d664ceea4
	github.com/stretchr/testify v1.6.1 // indirect
	go.etcd.io/bbolt v1.3.6-0.20200807205753-f6be82302843
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20201029221708-28c70e62bb1d
	golang.org/x/sys v0.0.0-20201029080932-201ba4db2418
	golang.org/x/text v0.3.4 // indirect
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/genproto v0.0.0-20201021134325-0d71844de594 // indirect
	google.golang.org/grpc v1.34.0
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/macaroon-bakery.v2 v2.0.1
	gopkg.in/macaroon.v2 v2.0.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
