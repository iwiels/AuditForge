package memory

import (
	"testing"

	"orquestador-auditor/internal/model"
)

func TestPersistAuditAndSearch(t *testing.T) {
	store := New(t.TempDir())
	state := model.NewAuditState("https://example.com", ".orquestador", model.TargetKindURL)
	state.Campaign = "q3-web"
	state.AddAsset(model.Asset{Type: "api-request", Value: "https://example.com/api/users"})
	state.AddFinding(model.Finding{Title: "Hidden admin endpoint", Severity: model.SeverityHigh, Description: "found via ffuf", ToolSource: "ffuf", Confirmed: false})
	if err := store.PersistAudit(state, "full"); err != nil {
		t.Fatalf("PersistAudit() error = %v", err)
	}
	items, err := store.Search("admin", 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected memory search results")
	}
	recent, err := store.Recent(10)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}
	if len(recent) == 0 {
		t.Fatalf("expected recent memory items")
	}
}
