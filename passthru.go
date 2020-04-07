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
// Author Adam Janikowski
//

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

var _ http.Handler = &passthru{}

func NewPassthru(arangodbEndpoint, jwt string, sslVerify bool, timeout time.Duration) (http.Handler, error) {
	transport := &http.Transport{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/_admin/metrics", arangodbEndpoint), nil)
	if err != nil {
		return nil, maskAny(err)
	}

	if !sslVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if jwt != "" {
		hdr, err := CreateArangodJwtAuthorizationHeader(jwt)
		if err != nil {
			return nil, maskAny(err)
		}
		req.Header.Add("Authorization", hdr)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &passthru{
		client:  client,
		request: req,
	}, nil
}

type passthru struct {
	request *http.Request
	client  *http.Client
}

func (p passthru) get() (*http.Response, error) {
	return p.client.Do(p.request)
}

func (p passthru) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	data, err := p.get()

	if err != nil {
		// Ignore error
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(err.Error()))
		return
	}

	if data.Body == nil {
		// Ignore error
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("Body is empty"))
		return
	}

	defer data.Body.Close()

	_, err = io.Copy(resp, data.Body)

	if err != nil {
		// Ignore error
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte("Unable to write body"))
		return
	}
}
