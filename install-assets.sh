#!/usr/bin/env bash
#
# orquestador-auditor - Assets Installer (Skills, Agents, MCP Config)
# Installs skills, agents, and MCP configuration for the Security Audit Orchestrator
#
# Usage:
#   ./install-assets.sh            # Interactive install (all)
#   ./install-assets.sh --skills   # Install skills only
#   ./install-assets.sh --agents   # Install agents only
#   ./install-assets.sh --mcp      # Install MCP config only
#   ./install-assets.sh --all      # Full install (default)
#   ./install-assets.sh --uninstall
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

# Get install directory (where this script is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Install locations
SKILLS_DIR="${SKILLS_DIR:-$HOME/.orquestador-auditor/skills}"
OPENCODE_DIR="${OPENCODE_DIR:-$HOME/.config/opencode}"
OPENCODE_AGENTS_DIR="$OPENCODE_DIR/agents"
OPENCODE_COMMANDS_DIR="$OPENCODE_DIR/commands"

# Features
INSTALL_SKILLS=true
INSTALL_AGENTS=true
INSTALL_MCP=true
UNINSTALL_MODE=false

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skills)
                INSTALL_SKILLS=true
                INSTALL_AGENTS=false
                INSTALL_MCP=false
                shift
                ;;
            --agents)
                INSTALL_SKILLS=false
                INSTALL_AGENTS=true
                INSTALL_MCP=false
                shift
                ;;
            --mcp)
                INSTALL_SKILLS=false
                INSTALL_AGENTS=false
                INSTALL_MCP=true
                shift
                ;;
            --all)
                INSTALL_SKILLS=true
                INSTALL_AGENTS=true
                INSTALL_MCP=true
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
                echo "  --all       Full install (default)"
                echo "  --uninstall Uninstall all assets"
                echo "  --help      Show this help"
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
    print_msg warning "Uninstalling orquestador-auditor assets..."

    [[ -d "$SKILLS_DIR" ]] && rm -rf "$SKILLS_DIR" && print_msg success "Removed: $SKILLS_DIR"
    [[ -d "$OPENCODE_AGENTS_DIR" ]] && rm -rf "$OPENCODE_AGENTS_DIR" && print_msg success "Removed: $OPENCODE_AGENTS_DIR"
    [[ -d "$OPENCODE_COMMANDS_DIR" ]] && rm -rf "$OPENCODE_COMMANDS_DIR" && print_msg success "Removed: $OPENCODE_COMMANDS_DIR"

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

    print_msg success "MCP config: $mcp_config"

    # Try to install MCP server
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

show_banner() {
    echo ""
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║     Orquestador Auditor - Assets Installer v2.0         ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

main() {
    local os=$(detect_os)

    show_banner

    print_msg info "Platform: $os"
    print_msg info "Project:  $PROJECT_ROOT"
    print_msg info "Target:   $OPENCODE_DIR"
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

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    print_msg success "Installation complete!"
    echo ""
    echo "  Next steps:"
    echo "    1. Restart your AI client (OpenCode/Claude Code)"
    echo "    2. Verify: /memory-search test"
    echo "    3. Start audit: /team https://target.com"
    echo ""
    echo "  Locations:"
    echo "    Skills:   $SKILLS_DIR"
    echo "    Agents:   $OPENCODE_AGENTS_DIR"
    echo "    Commands: $OPENCODE_COMMANDS_DIR"
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Run
parse_args "$@"
main
