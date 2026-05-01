#!/bin/bash
# AuditForge Proxy - Quick Setup Script
# Este script configura rápidamente el servidor proxy

set -e

echo "🚀 AuditForge Proxy Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Colores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Detectar OS
OS="unknown"
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    OS="windows"
fi

echo -e "${GREEN}✓${NC} Detected OS: $OS"

# Verificar Go
echo ""
echo "📦 Checking dependencies..."
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗${NC} Go is not installed. Please install Go 1.21+"
    exit 1
fi
echo -e "${GREEN}✓${NC} Go version: $(go version)"

# Verificar OpenSSL
if ! command -v openssl &> /dev/null; then
    echo -e "${RED}✗${NC} OpenSSL is not installed"
    exit 1
fi
echo -e "${GREEN}✓${NC} OpenSSL found"

# Crear directorios
echo ""
echo "📁 Creating directories..."
mkdir -p certs
mkdir -p logs
echo -e "${GREEN}✓${NC} Directories created"

# Generar certificados si no existen
echo ""
echo "🔐 Setting up CA certificates..."
if [ ! -f "certs/ca.crt" ]; then
    echo "   Generating CA certificate..."
    openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
        -keyout certs/ca.key -out certs/ca.crt \
        -subj "/CN=AuditForge Proxy CA/O=AuditForge/C=US" \
        -addext "basicConstraints=critical,CA:TRUE" 2>/dev/null
    
    echo -e "${GREEN}✓${NC} CA certificate generated"
else
    echo -e "${YELLOW}⚠${NC} CA certificate already exists"
fi

# Instalar certificado según OS
echo ""
echo "🔒 Installing CA certificate..."
echo -e "${YELLOW}⚠${NC} This may require sudo/admin privileges"

if [ "$OS" == "macos" ]; then
    echo "   Installing for macOS..."
    sudo security add-trusted-cert -d -r trustRoot \
        -k /Library/Keychains/System.keychain certs/ca.crt 2>/dev/null || \
        echo -e "${YELLOW}⚠${NC} Could not install certificate automatically"
    echo "   Manual: Open 'certs/ca.crt' in Keychain Access and trust it"
    
elif [ "$OS" == "linux" ]; then
    echo "   Installing for Linux..."
    sudo cp certs/ca.crt /usr/local/share/ca-certificates/auditforge.crt 2>/dev/null && \
        sudo update-ca-certificates 2>/dev/null || \
        echo -e "${YELLOW}⚠${NC} Could not install certificate automatically"
    echo "   Manual: Copy certs/ca.crt to /usr/local/share/ca-certificates/"
    
elif [ "$OS" == "windows" ]; then
    echo "   Installing for Windows..."
    if command -v powershell &> /dev/null; then
        powershell -Command "Import-Certificate -FilePath 'certs/ca.crt' -CertStoreLocation Cert:\\LocalMachine\\Root" 2>/dev/null || \
            echo -e "${YELLOW}⚠${NC} Could not install certificate automatically (need admin)"
    fi
    echo "   Manual: Right-click certs/ca.crt → Install Certificate → Local Machine → Trusted Root"
fi

# Instalar dependencias Go
echo ""
echo "📥 Installing Go dependencies..."
cd "$(dirname "$0")"

# Crear go.mod si no existe
if [ ! -f "go.mod" ]; then
    go mod init auditforge-proxy 2>/dev/null || true
fi

# Instalar dependencias
go get github.com/google/uuid 2>/dev/null || true
go get github.com/mark3labs/mcp-go 2>/dev/null || true
go get github.com/mattn/go-sqlite3 2>/dev/null || true

echo -e "${GREEN}✓${NC} Dependencies installed"

# Compilar
echo ""
echo "🔨 Building proxy server..."
go build -o auditforge-proxy .
echo -e "${GREEN}✓${NC} Binary built: ./auditforge-proxy"

# Crear script de inicio
echo ""
echo "📝 Creating startup scripts..."

cat > start-proxy.sh << 'EOF'
#!/bin/bash
# Start AuditForge Proxy Server

echo "🚀 Starting AuditForge Proxy Server..."
echo "   Proxy: http://localhost:8080"
echo "   Logs:  ./logs/proxy.log"
echo ""
echo "Configure your browser to use:"
echo "   HTTP Proxy:  localhost:8080"
echo "   HTTPS Proxy: localhost:8080"
echo ""
echo "Press Ctrl+C to stop"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

./auditforge-proxy 2>&1 | tee logs/proxy.log
EOF

chmod +x start-proxy.sh

# Script para Windows
cat > start-proxy.bat << 'EOF'
@echo off
echo 🚀 Starting AuditForge Proxy Server...
echo    Proxy: http://localhost:8080
echo    Logs:  .\logs\proxy.log
echo.
echo Configure your browser to use:
echo    HTTP Proxy:  localhost:8080
echo    HTTPS Proxy: localhost:8080
echo.
echo Press Ctrl+C to stop
echo ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

auditforge-proxy.exe
EOF

echo -e "${GREEN}✓${NC} Startup scripts created"

# Resumen
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo ""
echo "📋 Next Steps:"
echo ""
echo "1. Start the proxy server:"
echo "   ./start-proxy.sh          # Linux/macOS"
echo "   .\\start-proxy.bat         # Windows"
echo ""
echo "2. Configure your browser:"
echo "   HTTP Proxy:  localhost:8080"
echo "   HTTPS Proxy: localhost:8080"
echo ""
echo "3. In OpenCode, add to opencode.json:"
echo '   {'
echo '     "mcpServers": {'
echo '       "auditforge-proxy": {'
echo '         "command": "./auditforge-proxy"'
echo '       }'
echo '     }'
echo '   }'
echo ""
echo "4. Start auditing with:"
echo "   proxy.intercept.enable({})"
echo "   proxy.history.search({})"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📚 Documentation:"
echo "   ./README.md"
echo "   ../../internal/assets/skills/auditforge-proxy/SKILL.md"
echo ""
