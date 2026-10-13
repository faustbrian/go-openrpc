// Package httpstore provides an optional, SSRF-resistant HTTP document store
// for the reference resolver. Core reference parsing never imports or invokes
// this package implicitly. New clones http.DefaultTransport, including proxy/
// TLS policy; deployment must control ambient proxy configuration and transport
// customization. Direct DNS candidates are checked and dialed by IP; a proxy
// remains a separate trust/target-resolution boundary. Store.Load returns owned
// bytes and closes each response, but has no public idle-connection close API.
// Standard HTTP transport may create workers on use; no cgo is introduced here.
package httpstore

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrPolicy reports invalid time/byte/redirect policy, nil handle/context or
	// nonpositive read bound.
	ErrPolicy = errors.New("reference httpstore: invalid policy")
	// ErrURIDenied reports an unauthorized scheme/host, user information or fragment.
	ErrURIDenied = errors.New("reference httpstore: URI denied")
	// ErrAddressDenied reports a refused DNS/IP target without exposing its address.
	ErrAddressDenied = errors.New("reference httpstore: network address denied")
	// ErrRedirect reports redirected-target refusal or too many redirects.
	ErrRedirect = errors.New("reference httpstore: redirect denied")
	// ErrRequest reports sanitized DNS, transport or body-read failure.
	ErrRequest = errors.New("reference httpstore: request failed")
	// ErrHTTPStatus reports any status other than 200.
	ErrHTTPStatus = errors.New("reference httpstore: unexpected HTTP status")
	// ErrContentEncoding reports content encoding other than absent/identity.
	ErrContentEncoding = errors.New("reference httpstore: content encoding denied")
	// ErrResponseLimit reports an advertised or read body beyond maxBytes.
	ErrResponseLimit = errors.New("reference httpstore: response limit exceeded")
)

// Policy explicitly authorizes HTTP targets and bounds transport behavior.
type Policy struct {
	// AllowedHosts lists copied exact hostnames; matching lowercases and removes one
	// trailing dot, without constraining ports.
	AllowedHosts []string
	// AllowHTTP permits plaintext HTTP in addition to HTTPS.
	AllowHTTP bool
	// AllowPrivateAddresses disables the built-in nonpublic-IP refusal; deployment owns
	// the resulting trust boundary.
	AllowPrivateAddresses bool
	// MaxRedirects caps followed redirects; zero allows none.
	MaxRedirects int
	// Timeout bounds the complete HTTP exchange through the client.
	Timeout time.Duration
	// DialTimeout bounds each dial attempt, not all candidate addresses together.
	DialTimeout time.Duration
	// ResponseHeaderTimeout bounds waiting for response headers.
	ResponseHeaderTimeout time.Duration
	// MaxResponseHeaderBytes caps parsed response-header bytes through the transport.
	MaxResponseHeaderBytes int64
}

// DefaultPolicy permits only HTTPS, denies every host until allowlisted, and
// blocks private, loopback, link-local, multicast, and unspecified addresses.
func DefaultPolicy() Policy {
	return Policy{
		MaxRedirects:           3,
		Timeout:                15 * time.Second,
		DialTimeout:            5 * time.Second,
		ResponseHeaderTimeout:  5 * time.Second,
		MaxResponseHeaderBytes: 1 << 20,
	}
}

// Store is a bounded HTTP implementation of reference.Store.
type Store struct {
	client       *http.Client
	policy       Policy
	allowedHosts map[string]struct{}
	lookupIPAddr func(context.Context, string) ([]net.IPAddr, error)
}

// New constructs an isolated client with compression disabled, safe DNS/IP
// dialing, and bounded redirects. It requires http.DefaultTransport to be a
// nonnil *http.Transport; replacing it with another RoundTripper panics. Cloned
// proxy/TLS settings are caller-controlled. Construction itself starts no workers.
func New(policy Policy) (*Store, error) {
	if policy.MaxRedirects < 0 || policy.Timeout <= 0 || policy.DialTimeout <= 0 ||
		policy.ResponseHeaderTimeout <= 0 || policy.MaxResponseHeaderBytes <= 0 {
		return nil, ErrPolicy
	}
	policy.AllowedHosts = append([]string(nil), policy.AllowedHosts...)
	store := &Store{
		policy:       policy,
		allowedHosts: make(map[string]struct{}, len(policy.AllowedHosts)),
		lookupIPAddr: net.DefaultResolver.LookupIPAddr,
	}
	for _, host := range policy.AllowedHosts {
		host = strings.ToLower(strings.TrimSuffix(host, "."))
		if host == "" || strings.ContainsAny(host, "/:@") {
			return nil, ErrPolicy
		}
		store.allowedHosts[host] = struct{}{}
	}
	dialer := &net.Dialer{
		Timeout: policy.DialTimeout,
		// Use a literal duration so mutation testing does not manufacture an
		// equivalent multiplication of two compile-time constants.
		KeepAlive: time.Duration(30_000_000_000),
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableCompression = true
	transport.ResponseHeaderTimeout = policy.ResponseHeaderTimeout
	transport.MaxResponseHeaderBytes = policy.MaxResponseHeaderBytes
	transport.DialContext = store.dialContext(dialer)
	store.client = &http.Client{
		Transport: transport,
		Timeout:   policy.Timeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) > policy.MaxRedirects || store.authorize(request.URL) != nil {
				return ErrRedirect
			}
			return nil
		},
	}
	return store, nil
}

// Load fetches one status-200 identity-encoded resource into owned bytes. It
// does not parse JSON or assert Content-Type. Bounds must be positive and leave
// room for the extra sentinel byte in int64; use finite practical read limits.
// Responses are closed on every post-response path. Errors are sanitized; a
// cancellation during body reading can be returned as ErrRequest.
func (store *Store) Load(ctx context.Context, documentURI string, maxBytes int) ([]byte, error) {
	if store == nil || store.client == nil || ctx == nil || maxBytes <= 0 {
		return nil, ErrPolicy
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, err := url.Parse(documentURI)
	if err != nil || store.authorize(target) != nil {
		return nil, ErrURIDenied
	}
	// An authorized absolute HTTP(S) URL is always a valid client request URL.
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "identity")
	response, err := store.client.Do(request)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		switch {
		case errors.Is(err, ErrAddressDenied):
			return nil, ErrAddressDenied
		case errors.Is(err, ErrRedirect):
			return nil, ErrRedirect
		default:
			return nil, ErrRequest
		}
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		return nil, ErrHTTPStatus
	}
	encoding := strings.TrimSpace(strings.ToLower(response.Header.Get("Content-Encoding")))
	if encoding != "" && encoding != "identity" {
		return nil, ErrContentEncoding
	}
	if response.ContentLength > int64(maxBytes) {
		return nil, ErrResponseLimit
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(maxBytes)+1))
	if err != nil {
		return nil, ErrRequest
	}
	if len(data) > maxBytes {
		return nil, ErrResponseLimit
	}
	return data, nil
}

func (store *Store) authorize(target *url.URL) error {
	if target == nil {
		return ErrURIDenied
	}
	if target.User != nil {
		return ErrURIDenied
	}
	if target.Fragment != "" {
		return ErrURIDenied
	}
	if target.Scheme != "https" {
		if target.Scheme != "http" {
			return ErrURIDenied
		}
		if !store.policy.AllowHTTP {
			return ErrURIDenied
		}
	}
	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	if host == "" {
		return ErrURIDenied
	}
	if _, allowed := store.allowedHosts[host]; !allowed {
		return ErrURIDenied
	}
	return nil
}

func (store *Store) dialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, ErrAddressDenied
		}
		addresses, err := store.lookupIPAddr(ctx, host)
		if err != nil {
			return nil, ErrRequest
		}
		if len(addresses) == 0 {
			return nil, ErrRequest
		}
		for _, address := range addresses {
			if !store.policy.AllowPrivateAddresses && deniedAddress(address.IP) {
				return nil, ErrAddressDenied
			}
		}
		for _, address := range addresses {
			connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(address.IP.String(), port))
			if dialErr == nil {
				return connection, nil
			}
		}
		return nil, ErrRequest
	}
}

func deniedAddress(address net.IP) bool {
	if address.IsPrivate() {
		return true
	}
	if address.IsLoopback() {
		return true
	}
	if address.IsLinkLocalUnicast() {
		return true
	}
	if address.IsLinkLocalMulticast() {
		return true
	}
	if address.IsMulticast() {
		return true
	}
	return !address.IsGlobalUnicast()
}
