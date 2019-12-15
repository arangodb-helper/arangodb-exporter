//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"
	"os"

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
		Use: "arangodb-exporter",
		Run: cmdMainRun,
	}

	serverOptions   ServerConfig
	arangodbOptions struct {
		endpoint  string
		jwtSecret string
		jwtFile   string
		timeout   time.Duration
	}
)

func init() {
	f := cmdMain.Flags()

	f.StringVar(&serverOptions.Address, "server.address", ":9101", "Address the exporter will listen on (IP:port)")
	f.StringVar(&serverOptions.TLSKeyfile, "ssl.keyfile", "", "File containing TLS certificate used for the metrics server. Format equal to ArangoDB keyfiles")

	f.StringVar(&arangodbOptions.jwtSecret, "arangodb.jwtsecret", "", "JWT Secret used for authentication with ArangoDB server")
	f.StringVar(&arangodbOptions.jwtFile, "arangodb.jwt-file", "", "File containing the JWT for authentication with ArangoDB server")
	f.DurationVar(&arangodbOptions.timeout, "arangodb.timeout", time.Second*15, "Timeout of statistics requests for ArangoDB")

	f.MarkDeprecated("arangodb.jwtsecret", "please use --arangodb.jwt-file instead")
}

func main() {
	cmdMain.Execute()
}

func cmdMainRun(cmd *cobra.Command, args []string) {
	log.Infoln(fmt.Sprintf("Starting arangodb-exporter %s, build %s", projectVersion, projectBuild))

	var token string
	var conn string
        conn = os.Getenv("DB_URL")
	
	if arangodbOptions.jwtFile != "" {
		data, err := ioutil.ReadFile(arangodbOptions.jwtFile)
		if err != nil {
			log.Fatal(err)
		}
		token = strings.TrimSpace(string(data))
	} else if arangodbOptions.jwtSecret != "" {
		var err error
		token, err = CreateArangodJWT(arangodbOptions.jwtSecret)
		if err != nil {
			log.Fatal(err)
		}
	}

	exporter, err := NewExporter(conn, token, false, arangodbOptions.timeout)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	version.Version = projectVersion
	version.Revision = projectBuild
	prometheus.MustRegister(version.NewCollector("arangodb_exporter"))

	log.Infoln("Listening on", serverOptions.Address)

	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>ArangoDB Exporter</title></head>
             <body>
             <h1>ArangoDB Exporter</h1>
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})

	server, err := NewServer(mux, serverOptions)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(server.Run())
}
