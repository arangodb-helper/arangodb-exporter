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
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.2
	github.com/prometheus/common v0.3.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.1.0 // indirect
)
