package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"github.com/json-iterator/go"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktconfig/version"

	"github.com/decred/go-socks/socks"
	"github.com/pkt-cash/pktd/btcjson"
)

// newHTTPClient returns a new HTTP client that is configured according to the
// proxy and TLS settings in the associated connection configuration.
func newHTTPClient(cfg *config) (*http.Client, er.R) {
	// Configure proxy if needed.
	var dial func(network, addr string) (net.Conn, error)
	if cfg.Proxy != "" {
		proxy := &socks.Proxy{
			Addr:     cfg.Proxy,
			Username: cfg.ProxyUser,
			Password: cfg.ProxyPass,
		}
		dial = func(network, addr string) (net.Conn, error) {
			c, err := proxy.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}

	// Configure TLS if needed.
	var tlsConfig *tls.Config
	if cfg.TLS && cfg.RPCCert != "" {
		pem, err := ioutil.ReadFile(cfg.RPCCert)
		if err != nil {
			return nil, er.E(err)
		}

		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(pem)
		tlsConfig = &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
	}

	// Create and return the new HTTP client potentially configured with a
	// proxy and TLS.
	client := http.Client{
		Transport: &http.Transport{
			Dial:            dial,
			TLSClientConfig: tlsConfig,
		},
	}
	return &client, nil
}

// sendPostRequest sends the marshaled JSON-RPC command using HTTP-POST mode
// to the server described in the passed config struct.  It also attempts to
// unmarshal the response as a JSON-RPC response and returns either the result
// field or the error field depending on whether or not there is an error.
func sendPostRequest(marhsaledJSON []byte, cfg *config) (*btcjson.Response, er.R) {
	// Generate a request to the configured RPC server.
	protocol := "http"
	if cfg.TLS {
		protocol = "https"
	}
	url := protocol + "://" + cfg.RPCServer
	bodyReader := bytes.NewReader(marhsaledJSON)
	httpRequest, errr := http.NewRequest("POST", url, bodyReader)
	if errr != nil {
		return nil, er.E(errr)
	}
	httpRequest.Close = true
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-Pkt-RPC-Version", fmt.Sprintf("%d", version.AppMajorVersion()))

	// Configure basic access authorization.
	httpRequest.SetBasicAuth(cfg.RPCUser, cfg.RPCPassword)

	// Create the new HTTP client that is configured according to the user-
	// specified options and submit the request.
	httpClient, err := newHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	httpResponse, errr := httpClient.Do(httpRequest)
	if errr != nil {
		return nil, er.E(errr)
	}

	// Read the raw bytes and close the response.
	respBytes, errr := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if errr != nil {
		err = er.Errorf("error reading json reply: %v", errr)
		return nil, err
	}

	// Handle unsuccessful HTTP responses
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		// Generate a standard error to return if the server body is
		// empty.  This should not happen very often, but it's better
		// than showing nothing in case the target server has a poor
		// implementation.
		if len(respBytes) == 0 {
			return nil, er.Errorf("%d %s", httpResponse.StatusCode,
				http.StatusText(httpResponse.StatusCode))
		}
		additionalMessage := ""
		if _, err := os.Stat(defaultConfigFile); httpResponse.StatusCode == 401 && err == nil {
			additionalMessage = fmt.Sprintf(" (Try deleting %s)", defaultConfigFile)
		}
		return nil, er.Errorf("%s%s", respBytes, additionalMessage)
	}

	// Unmarshal the response.
	var resp btcjson.Response
	if err := er.E(jsoniter.Unmarshal(respBytes, &resp)); err != nil {
		return nil, err
	}

	return &resp, nil
}
