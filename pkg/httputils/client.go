package httputils

import (
	"crypto/tls"
	"net/http"
	"time"
)

var httpClient = &Client{}

const (
	timeout = time.Second * 120
)

type Client struct {
	cli *http.Client
}

func NewHttpCli() error {
	httpClient = &Client{
		cli: &http.Client{
			Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       timeout,
		},
	}
	return nil
}
