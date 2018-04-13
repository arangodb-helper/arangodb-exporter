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
	"crypto/tls"
	"net/http"
	_ "net/http/pprof"
	"time"

	certificates "github.com/arangodb-helper/go-certificates"
)

// ServerConfig settings for the Server
type ServerConfig struct {
	Address    string // Address to listen on
	TLSKeyfile string // Keyfile containing TLS certificate
}

// Server is the HTTPS server for the operator.
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new server, fetching/preparing a TLS certificate.
func NewServer(handler http.Handler, cfg ServerConfig) (*Server, error) {
	httpServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadTimeout:       time.Second * 30,
		ReadHeaderTimeout: time.Second * 15,
		WriteTimeout:      time.Second * 30,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if cfg.TLSKeyfile != "" {
		tlsConfig, err := createTLSConfig(cfg.TLSKeyfile)
		if err != nil {
			return nil, maskAny(err)
		}
		tlsConfig.BuildNameToCertificate()
		httpServer.TLSConfig = tlsConfig
	}

	return &Server{
		httpServer: httpServer,
	}, nil
}

// Run the server until the program stops.
func (s *Server) Run() error {
	if s.httpServer.TLSConfig != nil {
		if err := s.httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			return maskAny(err)
		}
	} else {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return maskAny(err)
		}
	}
	return nil
}

// createTLSConfig creates a TLS config from the given keyfile.
func createTLSConfig(keyfile string) (*tls.Config, error) {
	cert, err := certificates.LoadKeyFile(keyfile)
	if err != nil {
		return nil, maskAny(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
