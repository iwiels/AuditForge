package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.Println("🚀 Starting AuditForge Proxy Server...")

	// Verificar/generar certificados CA
	caCertPath := "./certs/ca.crt"
	caKeyPath := "./certs/ca.key"

	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		log.Println("⚠️  CA certificates not found. Run 'auditforge-proxy init-certs' first")
		log.Println("   This will generate the root CA certificate for HTTPS interception")
	}

	// Crear servidor proxy
	proxy, err := NewProxyServer("localhost:8080", caCertPath, caKeyPath)
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
	log.Println("   • proxy.replay.execute")
	log.Println("   • proxy.stats.get")
	log.Println("   • proxy.findings.list")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("")
	log.Println("🌐 Proxy listening on: http://localhost:8080")
	log.Println("   Configure your browser/app to use this proxy")
	log.Println("")

	// Manejar señales de cierre
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n🛑 Shutting down...")
		proxy.Stop()
		os.Exit(0)
	}()

	// Iniciar stdio server
	if err := server.ServeStdio(mcpServer); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
