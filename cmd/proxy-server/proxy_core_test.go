package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestNormalizeHostForCert tests the host normalization function.
func TestNormalizeHostForCert(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example.com", "example.com"},
		{"Example.Com", "example.com"},
		{" example.com ", "example.com"},
		{"example.com:8080", "example.com"},
		{"example.com:443", "example.com"},
		{"[::1]:8080", "::1"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"[::1]", "::1"},
		{"[2001:db8::1]", "2001:db8::1"},
		{"localhost", "localhost"},
		{"localhost:80", "localhost"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := normalizeHostForCert(tt.input); got != tt.expected {
			t.Errorf("normalizeHostForCert(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// TestRemoveHopByHeaders tests removal of hop-by-hop headers.
func TestRemoveHopByHeaders(t *testing.T) {
	header := make(http.Header)
	header.Set("Connection", "keep-alive")
	header.Set("Proxy-Authenticate", "Basic")
	header.Set("TE", "trailers")
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Upgrade", "websocket")
	header.Set("Proxy-Connection", "keep-alive")
	header.Set("Content-Type", "text/plain") // should remain
	header.Set("User-Agent", "test")        // should remain

	removeHopBy hopHeaders(header)

	if header.Get("Connection") != "" {
		t.Error("Connection header should be removed")
	}
	if header.Get("Proxy-Authenticate") != "" {
		t.Error("Proxy-Authenticate header should be removed")
	}
	if header.Get("TE") != "" {
		t.Error("TE header should be removed")
	}
	if header.Get("Transfer-Encoding") != "" {
		t.Error("Transfer-Encoding header should be removed")
	}
	if header.Get("Upgrade") != "" {
		t.Error("Upgrade header should be removed")
	}
	if header.Get("Proxy-Connection") != "" {
		t.Error("Proxy-Connection header should be removed")
	}
	if header.Get("Content-Type") != "text/plain" {
		t.Error("Content-Type header should remain")
	}
	if header.Get("User-Agent") != "test" {
		t.Error("User-Agent header should remain")
	}
}

// generateTestCA creates a temporary CA certificate and key for testing.
func generateTestCA(t *testing.T) (tls.Certificate, *x509.Certificate, crypto.Signer) {
	// Create a private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create a self-signed certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),

		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey(), privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privateKey,
		Leaf:        cert,
	}

	return tlsCert, cert, privateKey
}

// TestNewProxyServer tests creating a new proxy server instance.
func TestNewProxyServer(t *testing.T) {
	// Generate a test CA
	caCert, caLeaf, caKey := generateTestCA(t)

	// Write the CA cert and key to temporary files
	// For simplicity, we'll use the in-memory bytes to create a tls.Certificate and then
	// we'll create temporary files on disk.
	// However, NewProxyServer expects file paths. We'll create temporary files.
	certFile := "./test_ca.crt"
	keyFile := "./test_ca.key"

	// Encode the cert and key to PEM
	certPEM := tls.EncodePeerCertificate(caLeaf)
	keyPEM, err := x509.MarshalPKCS8PrivateKey(caKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	// Write files
	if err := os.WriteFile(certFile, certPEM, 0o644); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	defer func() {
		_ = os.Remove(certFile)
		_ = os.Remove(keyFile)
	}()

	// Create the proxy server
	proxy, err := NewProxyServer("127.0.0.1:0", certFile, keyFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if proxy == nil {
		t.Fatalf("Expected proxy to be not nil")
	}
	// Check that the CA was loaded
	if proxy.caLeaf == nil {
		t.Fatalf("Expected caLeaf to be set")
	}
	if proxy.leafKey == nil {
		t.Fatalf("Expected leafKey to be set")
	}
	// Clean up
	_ = proxy.Stop()
}

// TestInterceptorEnableDisable tests enabling and disabling the interceptor.
func TestInterceptorEnableDisable(t *testing.T) {
	i := NewInterceptor()
	if i.enabled {
		t.Fatalf("Expected interceptor to be disabled by default")
	}
	i.Enable()
	if !i.enabled {
		t.Fatalf("Expected interceptor to be enabled after Enable()")
	}
	i.Disable()
	if i.enabled {
		t.Fatalf("Expected interceptor to be disabled after Disable()")
	}
}

// TestInterceptorSetFilters tests setting filters.
func TestInterceptorSetFilters(t *testing.T) {
	i := NewInterceptor()
	filters := []InterceptFilter{
		{HostPattern: "example.com"},
		{PathPattern: "/api"},
		{MethodPattern: "POST"},
	}
	i.SetFilters(filters)
	if len(i.filters) != 3 {
		t.Fatalf("Expected 3 filters, got %d", len(i.filters))
	}
	// Check that the filters are set correctly
	if i.filters[0].HostPattern != "example.com" {
		t.Fatalf("Expected first filter HostPattern to be example.com")
	}
	if i.filters[1].PathPattern != "/api" {
		t.Fatalf("Expected second filter PathPattern to be /api")
	}
	if i.filters[2].MethodPattern != "POST" {
		t.Fatalf("Expected third filter MethodPattern to be POST")
	}
}

// TestInterceptorShouldIntercept tests the ShouldIntercept logic.
func TestInterceptorShouldIntercept(t *testing.T) {
	i := NewInterceptor()

	// No filters -> should intercept everything
	if !i.ShouldIntercept(&http.Request{}) {
		t.Fatalf("Expected to intercept when no filters are set")
	}
	i.Enable()
	if !i.ShouldIntercept(&http.Request{}) {
		t.Fatalf("Expected to intercept when enabled and no filters")
	}
	i.Disable()
	if i.ShouldIntercept(&http.Request{}) {
		t.Fatalf("Expected not to intercept when disabled")
	}

	// With filters
	i.Enable()
	i.SetFilters([]InterceptFilter{
		{HostPattern: "example.com"},
	})
	req := &http.Request{
		Host: "example.com",
		URL: &url.URL{Path: "/test"},
	}
	if !i.ShouldIntercept(req) {
		t.Fatalf("Expected to intercept request to example.com")
	}
	req.Host = "other.com"
	if i.ShouldIntercept(req) {
		t.Fatalf("Expected not to intercept request to other.com")
	}
	req.Host = "example.com"
	req.URL.Path = "/api"
	i.SetFilters([]InterceptFilter{
		{PathPattern: "/api"},
	})
	if !i.ShouldIntercept(req) {
		t.Fatalf("Expected to intercept request with path /api")
	}
	req.URL.Path = "/other"
	if i.ShouldIntercept(req) {
		t.Fatalf("Expected not to intercept request with path /other")
	}
}

// TestInterceptorActionChannel tests getting and sending action channels.
func TestInterceptorActionChannel(t *testing.T) {
	i := NewInterceptor()
	requestID := "test-request"

	// Get the channel for a request ID
	ch := i.GetActionChannel(requestID)
	if ch == nil {
		t.Fatalf("Expected action channel to be not nil")
	}
	// Sending an action should work
	action := &InterceptAction{Action: "forward"}
	if err := i.SendAction(requestID, action); err != nil {
		t.Fatalf("Expected no error sending action, got %v", err)
	}
	// Receiving the action
	select {
	case received := <-ch:
		if received.Action != "forward" {
			t.Fatalf("Expected action forward, got %s", received.Action)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Timeout waiting for action")
	}
	// After sending, the channel should be buffered (size 1) so we can send again?
	// Actually, the channel is buffered with size 1, so we can send another without blocking.
	// But the interceptor's SendAction does not check if the channel is full; it will block if the buffer is full.
	// We'll test that by trying to send two actions and then receiving two.
	// However, note that the interceptor's SendAction uses a mutex RLock and then sends to the channel.
	// The channel is created with buffer 1, so the first send will succeed, the second will block until a receive.
	// We'll test that in a goroutine.

	// Send two actions in a goroutine and receive them in another.
	done := make(chan bool)
	go func() {
		i.SendAction(requestID, &InterceptAction{Action: "drop"})
		i.SendAction(requestID, &InterceptAction{Action: "modify"})
		done <- true
	}()
	// Receive two actions
	action1 := <-ch
	if action1.Action != "drop" {
		t.Fatalf("Expected first action to be drop, got %s", action1.Action)
	}
	action2 := <-ch
	if action2.Action != "modify" {
		t.Fatalf("Expected second action to be modify, got %s", action2.Action)
	}
	<-done // wait for the sender to finish
}

// TestInterceptorSendActionInvalidID tests sending an action to a non-existent request.
func TestInterceptorSendActionInvalidID(t *testing.T) {
	i := NewInterceptor()
	requestID := "non-existent"
	action := &InterceptAction{Action: "forward"}
	err := i.SendAction(requestID, action)
	if err == nil {
		t.Fatalf("Expected error sending action to non-existent request")
	}
	if !strings.Contains(err.Error(), "not found or already processed") {
		t.Fatalf("Expected error message about request not found, got %v", err)
	}
}
