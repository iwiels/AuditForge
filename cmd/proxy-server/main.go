package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.SetOutput(os.Stderr)
	log.Println("🚀 Starting AuditForge Proxy Server...")

	port := strings.TrimSpace(os.Getenv("PROXY_PORT"))
	if port == "" {
		port = "8080"
	}
	addr := "localhost:" + port

	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = "./auditforge-proxy.db"
	}

	caCertPath := strings.TrimSpace(os.Getenv("PROXY_CA_CERT"))
	caKeyPath := strings.TrimSpace(os.Getenv("PROXY_CA_KEY"))
	if caCertPath == "" || caKeyPath == "" {
		baseDir := filepath.Dir(dbPath)
		if baseDir == "." || baseDir == "" {
			baseDir = "."
		}
		caDir := filepath.Join(baseDir, "certs")
		caCertPath = filepath.Join(caDir, "ca.crt")
		caKeyPath = filepath.Join(caDir, "ca.key")
	}
	if err := ensureCAFiles(caCertPath, caKeyPath); err != nil {
		log.Fatalf("Failed to prepare CA certificates: %v", err)
	}

	// Crear servidor proxy
	proxy, err := NewProxyServer(addr, caCertPath, caKeyPath, dbPath)
	if err != nil {
		log.Fatalf("Failed to create proxy server: %v", err)
	}

	// Iniciar proxy en background
	if err := proxy.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}

	// Crear MCP server
	mcpServer := server.NewMCPServer(
		"auditforge-proxy",
		"1.0.0",
		server.WithLogging(),
	)

	// Registrar tools
	tools := NewMCPServerTools(proxy)
	tools.RegisterTools(mcpServer)

	// Iniciar MCP server (stdio)
	log.Println("📡 MCP Server ready. Waiting for connections...")
	log.Println("")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🔧 MCP Tools Available:")
	log.Println("   • proxy.intercept.enable")
	log.Println("   • proxy.intercept.disable")
	log.Println("   • proxy.history.search")
	log.Println("   • proxy.request.get")
	log.Println("   • proxy.request.modify")
	log.Println("   • proxy.request.forward")
	log.Println("   • proxy.request.drop")
	log.Println("   • proxy.stats.get")
	log.Println("   • proxy.findings.list")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")
	log.Printf("🌐 Proxy listening on: http://%s", addr)
	log.Println("   Configure your browser/app to use this proxy")
	log.Println("")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ServeStdio(mcpServer)
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down proxy server...")
		if err := proxy.Stop(); err != nil {
			log.Printf("proxy stop error: %v", err)
		}
	case err := <-serveErr:
		if err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}
}

func ensureCAFiles(certPath, keyPath string) error {
	if _, err := os.Stat(certPath); err == nil {
		if _, keyErr := os.Stat(keyPath); keyErr == nil {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o755); err != nil {
		return err
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"AuditForge Proxy CA"},
			CommonName:   "AuditForge Proxy Root CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}
	keyOut, err := os.OpenFile(keyPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return err
	}
	return nil
}
