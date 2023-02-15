module github.com/arangodb-helper/arangodb-exporter

go 1.12

replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.1

require (
	github.com/arangodb-helper/go-certificates v0.0.0-20180821055445-9fca24fc2680
	github.com/arangodb/go-driver v0.0.0-20190430103524-b14f41496c3d
	github.com/coreos/go-semver v0.3.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/pavel-v-chernykh/keystore-go v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/common v0.26.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
