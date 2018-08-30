// Copyright (c) 2018 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

//------------------------------------------------------------------------------

// ClientCertConfig contains config fields for a client certificate.
type ClientCertConfig struct {
	CertFile string `json:"cert_file" yaml:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file"`
}

// Config contains configuration params for TLS.
type Config struct {
	Enabled            bool               `json:"enabled" yaml:"enabled"`
	RootCAsFile        string             `json:"root_cas_file" yaml:"root_cas_file"`
	InsecureSkipVerify bool               `json:"skip_cert_verify" yaml:"skip_cert_verify"`
	ClientCertificates []ClientCertConfig `json:"client_certs" yaml:"client_certs"`
}

// NewConfig creates a new Config with default values.
func NewConfig() Config {
	return Config{
		Enabled:            false,
		RootCAsFile:        "",
		InsecureSkipVerify: false,
		ClientCertificates: []ClientCertConfig{},
	}
}

//------------------------------------------------------------------------------

// Get returns a valid *tls.Config based on the configuration values of Config.
func (c *Config) Get() (*tls.Config, error) {
	var rootCAs *x509.CertPool
	if len(c.RootCAsFile) > 0 {
		caCert, err := ioutil.ReadFile(c.RootCAsFile)
		if err != nil {
			return nil, err
		}

		rootCAs = x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(caCert)
	}

	clientCerts := []tls.Certificate{}

	for _, pair := range c.ClientCertificates {
		keyPair, err := tls.LoadX509KeyPair(pair.CertFile, pair.KeyFile)
		if nil != err {
			return nil, err
		}
		clientCerts = append(clientCerts, keyPair)
	}

	return &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
		RootCAs:            rootCAs,
		Certificates:       clientCerts,
	}, nil
}

//------------------------------------------------------------------------------
