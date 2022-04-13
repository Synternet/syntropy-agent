package kubernetes

// Note: Many parts of code in this file is chopped from official k8s library

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	tokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type tokenRoundTripper struct {
	bearer string
	source oauth2.TokenSource
	rt     http.RoundTripper
}

// newTokenRoundTripper adds the provided bearer token to a request
// If tokenFile is non-empty, it is periodically read,
// and the last successfully read content is used as the bearer token.
func newTokenRoundTripper() (http.RoundTripper, error) {
	if len(tokenFile) == 0 {
		return nil, fmt.Errorf("token file missing")
	}
	source := &fileTokenSource{
		path:   tokenFile,
		period: time.Minute,
	}

	token, err := source.Token()
	if err != nil {
		return nil, err
	}
	caCert, err := ioutil.ReadFile(rootCAFile)
	if err != nil {
		return nil, fmt.Errorf("root certificate %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		}}
	return &tokenRoundTripper{token.AccessToken, source, transport}, nil
}

func (rt *tokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return rt.rt.RoundTrip(req)
	}

	token := rt.bearer
	if rt.source != nil {
		refreshedToken, err := rt.source.Token()
		if err == nil {
			token = refreshedToken.AccessToken
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return rt.rt.RoundTrip(req)
}

type fileTokenSource struct {
	path   string
	period time.Duration
}

func (ts *fileTokenSource) Token() (*oauth2.Token, error) {
	tokb, err := ioutil.ReadFile(ts.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file %q: %s", ts.path, err)
	}
	tok := strings.TrimSpace(string(tokb))
	if len(tok) == 0 {
		return nil, fmt.Errorf("read empty token from file %q", ts.path)
	}

	return &oauth2.Token{
		AccessToken: tok,
		Expiry:      time.Now().Add(ts.period),
	}, nil
}
