package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattn/go-sqlite3"
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
	db             *sql.DB
	storage        *ProxyStorage
	interceptor    *Interceptor
	mutex          sync.RWMutex
	interceptedReq map[string]*InterceptedRequest
	isRunning      bool
	logger         *log.Logger
}

// NewProxyServer crea una nueva instancia del proxy
func NewProxyServer(addr string, caCertPath, caKeyPath string) (*ProxyServer, error) {
	// Cargar certificado CA
	cert, err := tls.LoadX509KeyPair(caCertPath, caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	// Inicializar storage
	storage, err := NewProxyStorage("./auditforge-proxy.db")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &ProxyServer{
		addr:           addr,
		caCert:         cert,
		storage:        storage,
		interceptor:    NewInterceptor(),
		interceptedReq: make(map[string]*InterceptedRequest),
		logger:         log.New(os.Stdout, "[PROXY] ", log.LstdFlags),
	}, nil
}

// Start inicia el servidor proxy
func (p *ProxyServer) Start() error {
	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	p.isRunning = true
	p.logger.Printf("Proxy server listening on %s", p.addr)

	go func() {
		for p.isRunning {
			conn, err := listener.Accept()
			if err != nil {
				if p.isRunning {
					p.logger.Printf("Accept error: %v", err)
				}
				continue
			}
			go p.handleConnection(conn)
		}
	}()

	return nil
}

// Stop detiene el servidor proxy
func (p *ProxyServer) Stop() {
	p.isRunning = false
	p.storage.Close()
}

// handleConnection maneja una conexión entrante
func (p *ProxyServer) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Leer el primer byte para detectar si es HTTPS (CONNECT)
	buf := make([]byte, 1)
	if _, err := clientConn.Read(buf); err != nil {
		return
	}

	// Si es CONNECT, manejar HTTPS
	if buf[0] == 0x43 { // 'C' de CONNECT
		p.handleHTTPS(clientConn)
	} else {
		// HTTP normal
		p.handleHTTP(clientConn, buf)
	}
}

// handleHTTPS maneja conexiones HTTPS (MITM)
func (p *ProxyServer) handleHTTPS(clientConn net.Conn) {
	// Leer el método CONNECT completo
	reader := bufio.NewReader(io.MultiReader(bytes.NewReader([]byte{0x43}), clientConn))
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	host := req.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	// Responder 200 Connection Established
	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection established\r\n\r\n")

	// Realizar handshake TLS con el cliente usando certificado dinámico
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{p.caCert},
		InsecureSkipVerify: true,
	}

	tlsConn := tls.Server(clientConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	defer tlsConn.Close()

	// Conectar al servidor destino
	serverConn, err := tls.Dial("tcp", host, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer serverConn.Close()

	// Proxy bidireccional
	var wg sync.WaitGroup
	wg.Add(2)

	// Cliente -> Servidor
	go func() {
		defer wg.Done()
		p.proxyHTTPSRequest(tlsConn, serverConn)
	}()

	// Servidor -> Cliente
	go func() {
		defer wg.Done()
		io.Copy(tlsConn, serverConn)
	}()

	wg.Wait()
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
func (p *ProxyServer) handleHTTP(clientConn net.Conn, firstByte []byte) {
	reader := bufio.NewReader(io.MultiReader(bytes.NewReader(firstByte), clientConn))
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	reqResp := p.createRequestResponse(req)

	// Conectar al servidor destino
	host := req.Host
	if !strings.Contains(host, ":") {
		host += ":80"
	}

	serverConn, err := net.Dial("tcp", host)
	if err != nil {
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
	p.storage.SaveRequest(reqResp)

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
}

// forwardRequest envía el request al servidor y retorna la respuesta
func (p *ProxyServer) forwardRequest(reqResp *RequestResponse, req *http.Request, serverConn net.Conn, clientConn net.Conn) {
	start := time.Now()

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
	p.storage.SaveRequest(reqResp)

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
	url, _ := url.Parse(reqResp.URL)
	req, _ := http.NewRequest(reqResp.Method, reqResp.URL, bytes.NewReader(reqResp.RequestBody))
	req.URL = url
	req.Host = reqResp.Host

	for name, value := range reqResp.RequestHeaders {
		req.Header.Set(name, value)
	}

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
