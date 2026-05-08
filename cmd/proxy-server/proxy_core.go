package main

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RequestResponse representa un par request/response interceptado
type RequestResponse struct {
	ID              string            `json:"id"`
	Timestamp       time.Time         `json:"timestamp"`
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Host            string            `json:"host"`
	Path            string            `json:"path"`
	Query           string            `json:"query"`
	RequestHeaders  map[string]string `json:"request_headers"`
	RequestBody     []byte            `json:"request_body"`
	ResponseStatus  int               `json:"response_status"`
	ResponseHeaders map[string]string `json:"response_headers"`
	ResponseBody    []byte            `json:"response_body"`
	Duration        time.Duration     `json:"duration"`
	IsIntercepted   bool              `json:"is_intercepted"`
	InterceptAction string            `json:"intercept_action,omitempty"`
	Tags            []string          `json:"tags"`
}

// InterceptedRequest representa un request pausado esperando acción
type InterceptedRequest struct {
	ReqResp    *RequestResponse
	ClientConn net.Conn
	ServerConn net.Conn
	ResponseCh chan *http.Response
}

// ProxyServer es el servidor proxy MITM
type ProxyServer struct {
	addr           string
	caCert         tls.Certificate
	caLeaf         *x509.Certificate
	leafKey        crypto.Signer
	storage        *ProxyStorage
	interceptor    *Interceptor
	stateMu        sync.RWMutex
	mutex          sync.RWMutex
	listener       net.Listener
	interceptedReq map[string]*InterceptedRequest
	certMu         sync.RWMutex
	certCache      map[string]*tls.Certificate
	isRunning      bool
	logger         *log.Logger
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func newBufferedConn(conn net.Conn, reader *bufio.Reader) net.Conn {
	if reader == nil {
		return conn
	}
	return &bufferedConn{Conn: conn, reader: reader}
}

// removeHopHeaders removes hop-by-hop headers from the given Header.
// See RFC 2616 section 13.5.1 for the list of hop-by-hop headers.
func removeHopHeaders(header http.Header) {
	hopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
		"Proxy-Connection", // non-standard but common
	}

	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil {
		if c.reader.Buffered() > 0 {
			return c.reader.Read(p)
		}
		c.reader = nil
	}
	return c.Conn.Read(p)
}

// NewProxyServer crea una nueva instancia del proxy
func NewProxyServer(addr string, caCertPath, caKeyPath, dbPath string) (*ProxyServer, error) {
	// Cargar certificado CA
	cert, err := tls.LoadX509KeyPair(caCertPath, caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}
	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("failed to parse CA certificate chain")
	}
	caLeaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA leaf certificate: %w", err)
	}
	if _, ok := cert.PrivateKey.(crypto.Signer); !ok {
		return nil, fmt.Errorf("CA private key does not implement crypto.Signer")
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate leaf key: %w", err)
	}

	// Inicializar storage
	if strings.TrimSpace(dbPath) == "" {
		dbPath = "./auditforge-proxy.db"
	}
	storage, err := NewProxyStorage(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &ProxyServer{
		addr:           addr,
		caCert:         cert,
		caLeaf:         caLeaf,
		leafKey:        leafKey,
		storage:        storage,
		interceptor:    NewInterceptor(),
		interceptedReq: make(map[string]*InterceptedRequest),
		certCache:      make(map[string]*tls.Certificate),
		logger:         log.New(os.Stderr, "[PROXY] ", log.LstdFlags),
	}, nil
}

// Start inicia el servidor proxy
func (p *ProxyServer) Start() error {
	p.stateMu.Lock()
	if p.isRunning {
		p.stateMu.Unlock()
		return nil
	}
	p.stateMu.Unlock()

	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	p.stateMu.Lock()
	p.listener = listener
	p.isRunning = true
	p.stateMu.Unlock()
	p.logger.Printf("Proxy server listening on %s", p.addr)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if p.isRunningLocked() {
					p.logger.Printf("Accept error: %v", err)
				}
				if errors.Is(err, net.ErrClosed) || !p.isRunningLocked() {
					return
				}
				continue
			}
			go p.handleConnection(conn)
		}
	}()

	return nil
}

// Stop detiene el servidor proxy
func (p *ProxyServer) Stop() error {
	p.stateMu.Lock()
	if !p.isRunning && p.listener == nil {
		storage := p.storage
		p.stateMu.Unlock()
		if storage != nil {
			return storage.Close()
		}
		return nil
	}
	listener := p.listener
	storage := p.storage
	p.isRunning = false
	p.listener = nil
	p.stateMu.Unlock()

	var errs []error
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, err)
		}
	}
	if storage != nil {
		if err := storage.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func (p *ProxyServer) isRunningLocked() bool {
	p.stateMu.RLock()
	defer p.stateMu.RUnlock()
	return p.isRunning
}

// normalizeHostForCert devuelve el host normalizado para usar en un certificado (sin puerto, IPv6 sin corchetes).
func normalizeHostForCert(host string) string {
	// Eliminar espacios y convertir a minúsculas
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	// Si es una dirección IPv6 entre corchetes, extraer la dirección
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end != -1 {
			host = host[1:end]
		}
	}
	// Eliminar el puerto si está presente
	if host, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	// Si no hay puerto, devolver el host tal cual
	return host
}

func (p *ProxyServer) certificateForHost(host string) (*tls.Certificate, error) {
	host = normalizeHostForCert(host)
	if host == "" {
		return nil, fmt.Errorf("missing SNI host")
	}

	p.certMu.RLock()
	if cert, ok := p.certCache[host]; ok {
		p.certMu.RUnlock()
		return cert, nil
	}
	p.certMu.RUnlock()

	if p.caLeaf == nil || p.caCert.PrivateKey == nil {
		return nil, fmt.Errorf("CA certificate is not initialized")
	}
	if leaf, ok := p.caCert.PrivateKey.(crypto.Signer); ok {
		_ = leaf
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"AuditForge MITM"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{host},
		PublicKeyAlgorithm:    x509.ECDSA,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, p.caLeaf, p.leafKey.Public(), p.caCert.PrivateKey)
	if err != nil {
		return nil, err
	}

	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{der, p.caCert.Certificate[0]},
		PrivateKey:   p.leafKey,
		Leaf:         leaf,
	}

	p.certMu.Lock()
	p.certCache[host] = cert
	p.certMu.Unlock()

	return cert, nil
}

func (p *ProxyServer) dialUpstreamHTTP(host string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	return dialer.Dial("tcp", host)
}

func (p *ProxyServer) dialUpstreamTLS(host string) (net.Conn, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if host == "" {
		return nil, fmt.Errorf("missing upstream host")
	}
	serverName := host
	if parsedHost, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		serverName = parsedHost
	}
	return tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", host, &tls.Config{
		RootCAs:    rootCAs,
		ServerName: serverName,
	})
}

// handleConnection maneja una conexión entrante
func (p *ProxyServer) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	if strings.EqualFold(req.Method, http.MethodConnect) {
		p.handleHTTPS(clientConn, reader, req)
		return
	}

	p.handleHTTP(clientConn, req)
}

// handleHTTPS maneja conexiones HTTPS (MITM)
func (p *ProxyServer) handleHTTPS(clientConn net.Conn, reader *bufio.Reader, req *http.Request) {
	connectHost := req.Host
	if connectHost == "" {
		return
	}

	if !strings.Contains(connectHost, ":") {
		connectHost += ":443"
	}

	if _, err := fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		return
	}

	wrappedConn := newBufferedConn(clientConn, reader)
	tlsConn := tls.Server(wrappedConn, &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			serverName := strings.TrimSpace(hello.ServerName)
			if serverName == "" {
				serverName = strings.TrimSuffix(req.Host, ":443")
				if serverName == "" {
					serverName = strings.TrimSuffix(connectHost, ":443")
				}
			}
			cert, err := p.certificateForHost(serverName)
			if err != nil {
				return nil, err
			}
			return cert, nil
		},
	})
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	defer tlsConn.Close()

	serverConn, err := p.dialUpstreamTLS(connectHost)
	if err != nil {
		p.logger.Printf("upstream TLS dial failed for %s: %v", connectHost, err)
		return
	}
	defer serverConn.Close()

	p.proxyHTTPSRequest(tlsConn, serverConn)
}

// proxyHTTPSRequest procesa requests HTTPS
func (p *ProxyServer) proxyHTTPSRequest(clientConn, serverConn net.Conn) {
	reader := bufio.NewReader(clientConn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			return
		}

		reqResp := p.createRequestResponse(req)

		// Verificar si debe interceptarse
		if p.interceptor.ShouldIntercept(req) {
			p.handleInterceptedRequest(reqResp, req, clientConn, serverConn)
		} else {
			p.forwardRequest(reqResp, req, serverConn, clientConn)
		}
	}
}

// handleHTTP maneja requests HTTP
func (p *ProxyServer) handleHTTP(clientConn net.Conn, req *http.Request) {
	reqResp := p.createRequestResponse(req)

	host := req.Host
	if host == "" {
		p.sendErrorResponse(clientConn, http.StatusBadRequest, "Missing Host header")
		return
	}
	if !strings.Contains(host, ":") {
		host += ":80"
	}

	serverConn, err := p.dialUpstreamHTTP(host)
	if err != nil {
		p.logger.Printf("upstream HTTP dial failed for %s: %v", host, err)
		return
	}
	defer serverConn.Close()

	if p.interceptor.ShouldIntercept(req) {
		p.handleInterceptedRequest(reqResp, req, clientConn, serverConn)
	} else {
		p.forwardRequest(reqResp, req, serverConn, clientConn)
	}
}

// createRequestResponse crea un objeto RequestResponse desde un http.Request
func (p *ProxyServer) createRequestResponse(req *http.Request) *RequestResponse {
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(body))

	headers := make(map[string]string)
	for name, values := range req.Header {
		headers[name] = strings.Join(values, ", ")
	}

	return &RequestResponse{
		ID:             uuid.New().String(),
		Timestamp:      time.Now(),
		Method:         req.Method,
		URL:            req.URL.String(),
		Host:           req.Host,
		Path:           req.URL.Path,
		Query:          req.URL.RawQuery,
		RequestHeaders: headers,
		RequestBody:    body,
		Tags:           []string{},
	}
}

// handleInterceptedRequest pausa el request y espera acción
func (p *ProxyServer) handleInterceptedRequest(reqResp *RequestResponse, req *http.Request, clientConn, serverConn net.Conn) {
	reqResp.IsIntercepted = true

	intercepted := &InterceptedRequest{
		ReqResp:    reqResp,
		ClientConn: clientConn,
		ServerConn: serverConn,
		ResponseCh: make(chan *http.Response, 1),
	}

	p.mutex.Lock()
	p.interceptedReq[reqResp.ID] = intercepted
	p.mutex.Unlock()

	// Guardar en storage
	if err := p.storage.SaveRequest(reqResp); err != nil {
		p.logger.Printf("failed to save intercepted request: %v", err)
	}

	p.logger.Printf("[INTERCEPTED] %s %s (ID: %s)", req.Method, req.URL, reqResp.ID)

	// Esperar acción (bloqueante)
	select {
	case modifiedReq := <-p.interceptor.GetActionChannel(reqResp.ID):
		if modifiedReq.Action == "drop" {
			p.sendErrorResponse(clientConn, 403, "Request dropped by interceptor")
		} else if modifiedReq.Action == "forward" {
			p.forwardModifiedRequest(reqResp, modifiedReq, serverConn, clientConn)
		}
	case <-time.After(5 * time.Minute):
		// Timeout: forward automáticamente
		p.forwardRequest(reqResp, req, serverConn, clientConn)
	}

	p.mutex.Lock()
	delete(p.interceptedReq, reqResp.ID)
	p.mutex.Unlock()

	// Limpiar el canal de acción del interceptor para evitar fugas de goroutines
	p.interceptor.RemoveActionChannel(reqResp.ID)
}

// forwardRequest envía el request al servidor y retorna la respuesta
func (p *ProxyServer) forwardRequest(reqResp *RequestResponse, req *http.Request, serverConn net.Conn, clientConn net.Conn) {
	start := time.Now()

	// Eliminar encabezados hop-by-hop antes de reenviar
	removeHopHeaders(req.Header)

	// Enviar request al servidor
	if err := req.Write(serverConn); err != nil {
		p.sendErrorResponse(clientConn, 502, "Bad Gateway")
		return
	}

	// Leer respuesta
	respReader := bufio.NewReader(serverConn)
	resp, err := http.ReadResponse(respReader, req)
	if err != nil {
		p.sendErrorResponse(clientConn, 502, "Bad Gateway")
		return
	}
	defer resp.Body.Close()

	// Eliminar encabezados hop-by-hop de la respuesta antes de enviar al cliente
	removeHopHeaders(resp.Header)

	// Capturar respuesta
	body, _ := io.ReadAll(resp.Body)
	reqResp.ResponseStatus = resp.StatusCode
	reqResp.ResponseBody = body
	reqResp.Duration = time.Since(start)

	respHeaders := make(map[string]string)
	for name, values := range resp.Header {
		respHeaders[name] = strings.Join(values, ", ")
	}
	reqResp.ResponseHeaders = respHeaders

	// Guardar en storage
	if err := p.storage.SaveRequest(reqResp); err != nil {
		p.logger.Printf("failed to save request: %v", err)
	}

	// Enviar respuesta al cliente
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.Write(clientConn)
}

// forwardModifiedRequest envía un request modificado
func (p *ProxyServer) forwardModifiedRequest(reqResp *RequestResponse, action *InterceptAction, serverConn, clientConn net.Conn) {
	// Aplicar modificaciones
	if action.ModifiedBody != nil {
		reqResp.RequestBody = action.ModifiedBody
	}

	if action.ModifiedHeaders != nil {
		reqResp.RequestHeaders = action.ModifiedHeaders
	}

	reqResp.InterceptAction = action.Action

	// Reconstruir request
	url, err := url.Parse(reqResp.URL)
	if err != nil {
		p.logger.Printf("failed to parse URL: %v", err)
		p.sendErrorResponse(clientConn, 400, "Bad Request")
		return
	}
	req, err := http.NewRequest(reqResp.Method, reqResp.URL, bytes.NewReader(reqResp.RequestBody))
	if err != nil {
		p.logger.Printf("failed to create request: %v", err)
		p.sendErrorResponse(clientConn, 400, "Bad Request")
		return
	}
	req.URL = url
	req.Host = reqResp.Host

	for name, value := range reqResp.RequestHeaders {
		req.Header.Set(name, value)
	}

	// Eliminar encabezados hop-by-hop antes de reenviar
	removeHopHeaders(req.Header)

	p.forwardRequest(reqResp, req, serverConn, clientConn)
}

// sendErrorResponse envía una respuesta de error al cliente
func (p *ProxyServer) sendErrorResponse(conn net.Conn, status int, message string) {
	fmt.Fprintf(conn, "HTTP/1.1 %d %s\r\n", status, message)
	fmt.Fprintf(conn, "Content-Type: text/plain\r\n")
	fmt.Fprintf(conn, "Content-Length: %d\r\n", len(message))
	fmt.Fprintf(conn, "\r\n")
	fmt.Fprintf(conn, "%s", message)
}

// InterceptAction representa una acción sobre un request interceptado
type InterceptAction struct {
	RequestID       string
	Action          string // "forward", "drop", "modify"
	ModifiedBody    []byte
	ModifiedHeaders map[string]string
}

// Interceptor gestiona la lógica de interceptación
type Interceptor struct {
	enabled       bool
	filters       []InterceptFilter
	actionChans   map[string]chan *InterceptAction
	mutex         sync.RWMutex
}

// InterceptFilter define cuándo interceptar un request
type InterceptFilter struct {
	HostPattern   string
	PathPattern   string
	MethodPattern string
	ContainsBody  *bool
}

// NewInterceptor crea un nuevo interceptor
func NewInterceptor() *Interceptor {
	return &Interceptor{
		filters:     []InterceptFilter{},
		actionChans: make(map[string]chan *InterceptAction),
	}
}

// Enable activa la interceptación
func (i *Interceptor) Enable() {
	i.enabled = true
}

// Disable desactiva la interceptación
func (i *Interceptor) Disable() {
	i.enabled = false
}

// SetFilters establece filtros de interceptación
func (i *Interceptor) SetFilters(filters []InterceptFilter) {
	i.filters = filters
}

// ShouldIntercept determina si un request debe interceptarse
func (i *Interceptor) ShouldIntercept(req *http.Request) bool {
	if !i.enabled {
		return false
	}

	if len(i.filters) == 0 {
		return true // Interceptar todo si no hay filtros
	}

	for _, filter := range i.filters {
		if i.matchesFilter(req, filter) {
			return true
		}
	}

	return false
}

// matchesFilter verifica si un request coincide con un filtro
func (i *Interceptor) matchesFilter(req *http.Request, filter InterceptFilter) bool {
	if filter.HostPattern != "" && !strings.Contains(req.Host, filter.HostPattern) {
		return false
	}
	if filter.PathPattern != "" && !strings.Contains(req.URL.Path, filter.PathPattern) {
		return false
	}
	if filter.MethodPattern != "" && req.Method != filter.MethodPattern {
		return false
	}
	return true
}

// GetActionChannel obtiene el canal de acciones para un request
func (i *Interceptor) GetActionChannel(requestID string) chan *InterceptAction {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if _, exists := i.actionChans[requestID]; !exists {
		i.actionChans[requestID] = make(chan *InterceptAction, 1)
	}
	return i.actionChans[requestID]
}

// SendAction envía una acción a un request interceptado
func (i *Interceptor) SendAction(requestID string, action *InterceptAction) error {
	i.mutex.RLock()
	ch, exists := i.actionChans[requestID]
	i.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("request %s not found or already processed", requestID)
	}

	ch <- action
	return nil
}

// RemoveActionChannel elimina el canal de acciones para un request.
func (i *Interceptor) RemoveActionChannel(requestID string) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	delete(i.actionChans, requestID)
}
