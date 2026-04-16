package cli

import (
	"flag"
	"fmt"
	"path/filepath"

	"orquestador-auditor/internal/memory"
)

func runMemory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("memory requires a subcommand: search or context")
	}
	switch args[0] {
	case "search":
		return runMemorySearch(args[1:])
	case "context":
		return runMemoryContext(args[1:])
	default:
		return fmt.Errorf("unknown memory subcommand %q", args[0])
	}
}

func runMemorySearch(args []string) error {
	fs := flag.NewFlagSet("memory search", flag.ContinueOnError)
	var dir string
	var query string
	var limit int
	fs.StringVar(&dir, "dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	fs.StringVar(&query, "query", "", "Search query")
	fs.IntVar(&limit, "limit", 10, "Maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if query == "" {
		return fmt.Errorf("memory search requires --query")
	}
	items, err := memory.New(dir).Search(query, limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Printf("[%s] %s\n%s\n\n", item.Kind, item.Title, item.Body)
	}
	return nil
}

func runMemoryContext(args []string) error {
	fs := flag.NewFlagSet("memory context", flag.ContinueOnError)
	var dir string
	var limit int
	fs.StringVar(&dir, "dir", filepath.Join(".orquestador", "memory"), "Memory directory")
	fs.IntVar(&limit, "limit", 10, "Maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	items, err := memory.New(dir).Recent(limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		fmt.Printf("[%s] %s\n%s\n\n", item.Kind, item.Title, item.Body)
	}
	return nil
}
