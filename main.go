package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/spf13/cobra"
)

var (
	projectVersion = "dev"
	projectBuild   = "dev"
	maskAny        = errors.WithStack

	cmdMain = &cobra.Command{
		Use: "arangodb_exporter",
		Run: cmdMainRun,
	}

	serverOptions struct {
		listenAddress string
	}
	arangodbOptions struct {
		endpoint  string
		jwtSecret string
		timeout   time.Duration
	}
)

func init() {
	f := cmdMain.Flags()

	f.StringVar(&serverOptions.listenAddress, "server.address", ":9101", "Address the exporter will listen on (IP:port)")

	f.StringVar(&arangodbOptions.endpoint, "arangodb.endpoint", "http://127.0.0.1:8529", "Endpoint used to reach the ArangoDB server")
	f.StringVar(&arangodbOptions.jwtSecret, "arangodb.jwtsecret", "", "JWT Secret used for authentication with ArangoDB server")
	f.DurationVar(&arangodbOptions.timeout, "arangodb.timeout", time.Second*15, "Timeout of statistics requests for ArangoDB")
}

func main() {
	cmdMain.Execute()
}

func cmdMainRun(cmd *cobra.Command, args []string) {
	log.Infoln("Starting arangodb_exporter %s, build %s", projectVersion, projectBuild)

	exporter, err := NewExporter(arangodbOptions.endpoint, arangodbOptions.jwtSecret, false, arangodbOptions.timeout)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("arangodb_exporter"))

	log.Infoln("Listening on", serverOptions.listenAddress)
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>ArangoDB Exporter</title></head>
             <body>
             <h1>ArangoDB Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(serverOptions.listenAddress, nil))
}
