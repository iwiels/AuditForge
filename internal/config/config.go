package config

import (
	"flag"
	"fmt"
	"strings"
)

type Config struct {
	Agent  string
	Bundle string
	All    bool
	MCP    bool
}

func Default() Config {
	return Config{}
}

func Parse(args []string) (Config, error) {
	cfg := Default()
	fs := flag.NewFlagSet("orquestador-auditor", flag.ContinueOnError)
	fs.StringVar(&cfg.Agent, "agent", "", "Agent to sync: claude-code, claude, cursor, opencode")
	fs.StringVar(&cfg.Bundle, "bundle", "", "Bundle to install: core-web, supply-chain, advanced-web, full")
	fs.BoolVar(&cfg.All, "all", false, "Sync all detected agents")
	fs.BoolVar(&cfg.MCP, "mcp", false, "Run as an MCP server")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	cfg.Agent = strings.TrimSpace(cfg.Agent)
	cfg.Bundle = strings.TrimSpace(cfg.Bundle)
	return cfg, nil
}

func Validate(cfg Config) error {
	if cfg.MCP {
		return nil
	}
	return fmt.Errorf("use a subcommand: install, sync, self-update, or ui")
}
