package exampleagent_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// Route the public API origin to a disposable upstream while retaining actual
// HTTP framing and cancellation. Production needs no test endpoint setting.
func hnHTTP(t *testing.T, handler http.Handler) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	target, err := url.Parse(server.URL)
	mustHN(t, err)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	t.Cleanup(func() { transport.CloseIdleConnections(); server.Close() })
	return &http.Client{Transport: hnRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		local := request.Clone(request.Context())
		endpoint := *request.URL
		endpoint.Scheme, endpoint.Host = target.Scheme, target.Host
		local.URL = &endpoint
		return transport.RoundTrip(local)
	})}
}

type hnRoundTripFunc func(*http.Request) (*http.Response, error)

func (f hnRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
