#!/usr/bin/env bash
#
# AuditForge - Assets Installer v2.0
# Installs skills, agents, MCP config, and builds the proxy server
#
# Usage:
#   ./install-assets.sh            # Install everything
#   ./install-assets.sh --skills   # Skills only
#   ./install-assets.sh --agents   # Agents only
#   ./install-assets.sh --mcp      # MCP config only
#   ./install-assets.sh --proxy    # Proxy server only
#   ./install-assets.sh --all      # Full install (default)
#   ./install-assets.sh --help
#
# Supports: Linux, macOS, Windows (Git Bash, WSL)
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Detect OS
detect_os() {
    local os_type="unknown"
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        os_type="linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        os_type="macos"
    elif [[ -d "/proc/version" ]] && grep -qi "microsoft" /proc/version 2>/dev/null; then
        os_type="wsl"
    elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
        os_type="windows"
    fi
    echo "$os_type"
}

# Get install directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Install locations
SKILLS_DIR="${SKILLS_DIR:-$HOME/.auditforge/skills}"
OPENCODE_DIR="${OPENCODE_DIR:-$HOME/.config/opencode}"
AUDITFORGE_DIR="${AUDITFORGE_DIR:-$HOME/.auditforge}"
PROXY_INSTALL_DIR="$AUDITFORGE_DIR/proxy"
OPENCODE_AGENTS_DIR="$OPENCODE_DIR/agents"
OPENCODE_COMMANDS_DIR="$OPENCODE_DIR/commands"

# Features
INSTALL_SKILLS=true
INSTALL_AGENTS=true
INSTALL_MCP=true
INSTALL_PROXY=true
UNINSTALL_MODE=false

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skills)
                INSTALL_SKILLS=true
                INSTALL_AGENTS=false
                INSTALL_MCP=false
                INSTALL_PROXY=false
                shift
                ;;
            --agents)
                INSTALL_SKILLS=false
                INSTALL_AGENTS=true
                INSTALL_MCP=false
                INSTALL_PROXY=false
                shift
                ;;
            --mcp)
                INSTALL_SKILLS=false
                INSTALL_AGENTS=false
                INSTALL_MCP=true
                INSTALL_PROXY=false
                shift
                ;;
            --proxy)
                INSTALL_SKILLS=false
                INSTALL_AGENTS=false
                INSTALL_MCP=false
                INSTALL_PROXY=true
                shift
                ;;
            --all)
                INSTALL_SKILLS=true
                INSTALL_AGENTS=true
                INSTALL_MCP=true
                INSTALL_PROXY=true
                shift
                ;;
            --uninstall)
                UNINSTALL_MODE=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skills    Install skills only"
                echo "  --agents    Install agents only"
                echo "  --mcp       Install MCP config only"
                echo "  --proxy     Build and install proxy server only"
                echo "  --all       Full install (default)"
                echo "  --uninstall Uninstall all assets"
                echo "  --help      Show this help"
                echo ""
                echo "This script will:"
                echo "  1. Install 29 security skills"
                echo "  2. Install agent definitions"
                echo "  3. Configure MCP servers"
                echo "  4. Build the AuditForge Proxy (requires Go)"
                echo "  5. Generate SSL certificates for HTTPS interception"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done
}

print_msg() {
    local type=$1
    local msg=$2
    local prefix=""
    case $type in
        info)    prefix="${BLUE}[INFO]${NC}" ;;
        success) prefix="${GREEN}[OK]${NC}" ;;
        warning) prefix="${YELLOW}[WARN]${NC}" ;;
        error)   prefix="${RED}[ERROR]${NC}" ;;
        step)    prefix="${CYAN}[STEP]${NC}" ;;
    esac
    echo -e "$prefix $msg"
}

check_deps() {
    local missing=()
    command -v git &> /dev/null || missing+=("git")
    command -v curl &> /dev/null || missing+=("curl")

    if [[ ${#missing[@]} -gt 0 ]]; then
        print_msg error "Missing dependencies: ${missing[*]}"
        exit 1
    fi
}

ensure_dir() {
    if [[ ! -d "$1" ]]; then
        mkdir -p "$1"
    fi
}

copy_file() {
    local src=$1
    local dst=$2
    local relative_path=${src#$PROJECT_ROOT/}

    ensure_dir "$(dirname "$dst")"

    if [[ -f "$dst" ]]; then
        cp "${dst}.backup.$(date +%Y%m%d%H%M%S)" <<< "$(cat "$dst")" 2>/dev/null || true
        print_msg warning "Backed up: $dst"
    fi

    cp "$src" "$dst"
    print_msg success "Installed: $relative_path"
}

uninstall() {
    print_msg warning "Uninstalling AuditForge assets..."

    [[ -d "$SKILLS_DIR" ]] && rm -rf "$SKILLS_DIR" && print_msg success "Removed: $SKILLS_DIR"
    [[ -d "$OPENCODE_AGENTS_DIR" ]] && rm -rf "$OPENCODE_AGENTS_DIR" && print_msg success "Removed: $OPENCODE_AGENTS_DIR"
    [[ -d "$OPENCODE_COMMANDS_DIR" ]] && rm -rf "$OPENCODE_COMMANDS_DIR" && print_msg success "Removed: $OPENCODE_COMMANDS_DIR"
    [[ -d "$PROXY_INSTALL_DIR" ]] && rm -rf "$PROXY_INSTALL_DIR" && print_msg success "Removed: $PROXY_INSTALL_DIR"

    print_msg success "Uninstall complete"
}

install_skills() {
    print_msg step "Installing skills..."

    ensure_dir "$SKILLS_DIR"

    local skills_src="$PROJECT_ROOT/internal/assets/skills"
    if [[ ! -d "$skills_src" ]]; then
        print_msg error "Skills directory not found: $skills_src"
        return 1
    fi

    local count=0
    for skill_dir in "$skills_src"/*/; do
        if [[ -d "$skill_dir" ]]; then
            local skill_name=$(basename "$skill_dir")
            local skill_file="$skill_dir/SKILL.md"

            if [[ -f "$skill_file" ]]; then
                ensure_dir "$SKILLS_DIR/$skill_name"
                cp "$skill_file" "$SKILLS_DIR/$skill_name/"
                count=$((count + 1))
                echo -en "  ${GREEN}✓${NC} $skill_name\n"
            fi
        fi
    done

    # Copy skill scripts
    if [[ -d "$skills_src/scripts" ]]; then
        ensure_dir "$SKILLS_DIR/scripts"
        cp -r "$skills_src/scripts/"* "$SKILLS_DIR/scripts/" 2>/dev/null || true
    fi

    echo ""
    print_msg success "Installed $count skills to $SKILLS_DIR"
}

install_agents() {
    print_msg step "Installing agents and commands..."

    ensure_dir "$OPENCODE_AGENTS_DIR"
    ensure_dir "$OPENCODE_COMMANDS_DIR"

    local count=0

    # OpenCode agents
    local opencode_agents="$PROJECT_ROOT/internal/assets/opencode/agents"
    if [[ -d "$opencode_agents" ]]; then
        for agent_file in "$opencode_agents"/*.md; do
            if [[ -f "$agent_file" ]]; then
                local agent_name=$(basename "$agent_file" .md)
                copy_file "$agent_file" "$OPENCODE_AGENTS_DIR/$agent_name.md"
                count=$((count + 1))
                echo -en "  ${GREEN}✓${NC} agent:$agent_name\n"
            fi
        done
    fi

    # OpenCode commands
    local opencode_cmds="$PROJECT_ROOT/internal/assets/opencode/commands"
    if [[ -d "$opencode_cmds" ]]; then
        for cmd_file in "$opencode_cmds"/*.md; do
            if [[ -f "$cmd_file" ]]; then
                local cmd_name=$(basename "$cmd_file" .md)
                copy_file "$cmd_file" "$OPENCODE_COMMANDS_DIR/$cmd_name.md"
                count=$((count + 1))
                echo -en "  ${GREEN}✓${NC} cmd:$cmd_name\n"
            fi
        done
    fi

    echo ""
    print_msg success "Installed $count agents/commands to $OPENCODE_DIR"
}

install_mcp() {
    print_msg step "Installing MCP configuration..."

    local os=$(detect_os)

    case $os in
        linux|macos|wsl)
            ensure_dir "$OPENCODE_DIR/mcp"
            ;;
        windows)
            local appdata="${APPDATA:-}"
            if [[ -n "$appdata" ]]; then
                OPENCODE_DIR="$appdata/opencode"
            fi
            ensure_dir "$OPENCODE_DIR/mcp"
            ;;
    esac

    # Chrome DevTools MCP
    local mcp_config="$OPENCODE_DIR/mcp/chrome-devtools.json"
    cat > "$mcp_config" << 'MCP_EOF'
{
  "mcpServers": {
    "chrome-devtools": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-chrome-devtools"]
    }
  }
}
MCP_EOF
    print_msg success "MCP config: chrome-devtools.json"

    # AuditForge Proxy MCP
    local proxy_mcp_config="$OPENCODE_DIR/mcp/auditforge-proxy.json"
    cat > "$proxy_mcp_config" << PROXY_MCP_EOF
{
  "mcpServers": {
    "auditforge-proxy": {
      "command": "$PROXY_INSTALL_DIR/auditforge-proxy",
      "args": [],
      "env": {
        "PROXY_PORT": "8080",
        "DB_PATH": "$PROXY_INSTALL_DIR/auditforge-proxy.db"
      }
    }
  }
}
PROXY_MCP_EOF
    print_msg success "MCP config: auditforge-proxy.json"

    # Try to install MCP servers
    if command -v npx &> /dev/null; then
        print_msg info "Verifying chrome-devtools MCP server..."
        if npx -y @modelcontextprotocol/server-chrome-devtools --version &> /dev/null; then
            print_msg success "MCP server ready"
        else
            print_msg warning "MCP server not available - will be installed on first use"
        fi
    fi

    echo ""
    print_msg success "MCP configuration installed"
}

install_proxy() {
    print_msg step "Building AuditForge Proxy Server..."

    local proxy_src="$PROJECT_ROOT/cmd/proxy-server"
    
    ensure_dir "$PROXY_INSTALL_DIR"
    ensure_dir "$PROXY_INSTALL_DIR/certs"

    # Check for Go
    if ! command -v go &> /dev/null; then
        print_msg warning "Go not found. Proxy server will not be built."
        print_msg warning "Install Go from https://golang.org/dl/ and re-run with --proxy"
        echo ""
        return 0
    fi

    print_msg info "Go version:"
    go version | sed 's/^/  /'

    # Check for OpenSSL
    local has_openssl=false
    if command -v openssl &> /dev/null; then
        has_openssl=true
    fi

    # Build proxy
    print_msg info "Building proxy server..."
    cd "$proxy_src"

    # Initialize go.mod if not exists
    if [[ ! -f "go.mod" ]]; then
        go mod init auditforge-proxy 2>/dev/null || true
        go get github.com/google/uuid 2>/dev/null || true
        go get github.com/mark3labs/mcp-go 2>/dev/null || true
        go get github.com/mattn/go-sqlite3 2>/dev/null || true
    fi

    # Build
    if go build -o "$PROXY_INSTALL_DIR/auditforge-proxy" .; then
        print_msg success "Proxy built: $PROXY_INSTALL_DIR/auditforge-proxy"
    else
        print_msg error "Failed to build proxy server"
        print_msg info "You may need to install a C compiler for SQLite3 support"
        print_msg info "Or download pre-built binaries from releases"
        cd "$PROJECT_ROOT"
        return 0
    fi

    # Generate certificates
    print_msg info "Generating SSL certificates..."
    
    if [[ "$has_openssl" == true ]]; then
        if openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
            -keyout "$PROXY_INSTALL_DIR/certs/ca.key" \
            -out "$PROXY_INSTALL_DIR/certs/ca.crt" \
            -subj "/CN=AuditForge Proxy CA" \
            -addext "basicConstraints=critical,CA:TRUE" 2>/dev/null; then
            print_msg success "CA certificate generated"
        else
            print_msg warning "Failed to generate CA certificate"
        fi
    else
        print_msg warning "OpenSSL not available - certificates must be generated manually"
    fi

    # Copy setup script
    if [[ -f "$proxy_src/setup.sh" ]]; then
        cp "$proxy_src/setup.sh" "$PROXY_INSTALL_DIR/"
        chmod +x "$PROXY_INSTALL_DIR/setup.sh"
    fi

    # Create start script
    cat > "$PROXY_INSTALL_DIR/start-proxy.sh" << 'START_SCRIPT'
#!/bin/bash
# Start AuditForge Proxy Server

echo "🚀 Starting AuditForge Proxy Server..."
echo "   Proxy: http://localhost:8080"
echo "   Press Ctrl+C to stop"
echo ""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$SCRIPT_DIR/auditforge-proxy"
START_SCRIPT
    chmod +x "$PROXY_INSTALL_DIR/start-proxy.sh"

    print_msg success "Start script created: start-proxy.sh"
    print_msg success "Proxy server installed to $PROXY_INSTALL_DIR"
}

show_banner() {
    echo ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║     AuditForge - Assets Installer v2.0                  ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

show_completion() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    print_msg success "Installation complete!"
    echo ""
    echo "  Next steps:"
    echo "    1. Restart your AI client (OpenCode/Claude Code)"
    echo "    2. Configure browser to use proxy localhost:8080"
    echo "    3. Start audit: /team https://target.com"
    echo ""
    echo "  Locations:"
    echo "    Skills:   $SKILLS_DIR"
    echo "    Agents:   $OPENCODE_AGENTS_DIR"
    echo "    Commands: $OPENCODE_COMMANDS_DIR"
    echo "    MCP:      $OPENCODE_DIR/mcp"
    echo "    Proxy:    $PROXY_INSTALL_DIR"
    echo ""
    
    if [[ -f "$PROXY_INSTALL_DIR/auditforge-proxy" ]]; then
        echo "  🎯 Proxy server is ready to use!"
        echo "     To start: $PROXY_INSTALL_DIR/start-proxy.sh"
        echo ""
		 echo "  🔧 MCP Tools Available:"
		 echo "     • proxy.intercept.enable"
		 echo "     • proxy.history.search"
		 echo "     • proxy.request.modify"
		 echo "     • proxy.findings.list"
        echo ""
    else
        echo "  ⚠️  Proxy server not built. Install Go and re-run with --proxy"
    fi
    
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

main() {
    local os=$(detect_os)

    show_banner

    print_msg info "Platform: $os"
    print_msg info "Project:  $PROJECT_ROOT"
    print_msg info "Target:   $OPENCODE_DIR"
    print_msg info "Proxy:    $PROXY_INSTALL_DIR"
    echo ""

    check_deps

    if [[ "$UNINSTALL_MODE" == "true" ]]; then
        uninstall
        exit 0
    fi

    echo ""
    if [[ "$INSTALL_SKILLS" == "true" ]]; then
        install_skills
        echo ""
    fi

    if [[ "$INSTALL_AGENTS" == "true" ]]; then
        install_agents
        echo ""
    fi

    if [[ "$INSTALL_MCP" == "true" ]]; then
        install_mcp
        echo ""
    fi

    if [[ "$INSTALL_PROXY" == "true" ]]; then
        install_proxy
        echo ""
    fi

    show_completion
}

# Run
parse_args "$@"
main
