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
		resp.Write([]byte(err.Error()))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	if data.Body == nil {
		// Ignore error
		resp.Write([]byte("Body is empty"))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer data.Body.Close()

	_, err = io.Copy(resp, data.Body)

	if data.Body == nil {
		// Ignore error
		resp.Write([]byte("Unable to write body"))
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}
