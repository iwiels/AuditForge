// Package memory provides a lightweight persistent store for security observations,
// following the Engram convention: observations are indexed by kind, title, body,
// target, and tags, and are searchable via SQLite FTS5 with a JSON file fallback.
package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Observation is a single persisted memory entry.
type Observation struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Target    string            `json:"target"`
	Campaign  string            `json:"campaign,omitempty"`
	RunID     string            `json:"run_id"`
	CreatedAt time.Time         `json:"created_at"`
	Tags      []string          `json:"tags,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Store is the Engram-style memory backend.
type Store struct {
	Root string
}

const databaseFilename = "memory.db"

// New returns a Store rooted at the given directory.
func New(root string) Store {
	return Store{Root: root}
}

// Save persists a single observation to the store.
func (s Store) Save(obs Observation) error {
	if err := os.MkdirAll(filepath.Join(s.Root, "observations"), 0o755); err != nil {
		return err
	}
	if obs.ID == "" {
		return fmt.Errorf("observation ID must not be empty")
	}
	if obs.CreatedAt.IsZero() {
		obs.CreatedAt = time.Now().UTC()
	}
	if err := writeJSON(filepath.Join(s.Root, "observations", obs.ID+".json"), obs); err != nil {
		return err
	}
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return s.upsertObservation(db, obs)
}

// Search finds observations matching the query string using FTS5, with JSON fallback.
func (s Store) Search(query string, limit int) ([]Observation, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return s.Recent(limit)
	}
	db, err := s.openDB()
	if err == nil {
		defer db.Close()
		rows, err := db.Query(`
			SELECT o.id, o.kind, o.title, o.body, o.target, o.campaign, o.run_id, o.created_at, o.tags_json, o.metadata_json
			FROM observations_fts f
			JOIN observations o ON o.id = f.id
			WHERE observations_fts MATCH ?
			ORDER BY bm25(observations_fts), o.created_at DESC
			LIMIT ?
		`, escapeFTSQuery(query), normalizeLimit(limit))
		if err == nil {
			defer rows.Close()
			return scanObservations(rows)
		}
	}
	return s.searchJSONFallback(query, limit)
}

// Recent returns the most recently created observations.
func (s Store) Recent(limit int) ([]Observation, error) {
	db, err := s.openDB()
	if err == nil {
		defer db.Close()
		rows, err := db.Query(`
			SELECT id, kind, title, body, target, campaign, run_id, created_at, tags_json, metadata_json
			FROM observations
			ORDER BY created_at DESC
			LIMIT ?
		`, normalizeLimit(limit))
		if err == nil {
			defer rows.Close()
			return scanObservations(rows)
		}
	}
	return s.recentJSONFallback(limit)
}

func (s Store) openDB() (*sql.DB, error) {
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(s.Root, databaseFilename))
	if err != nil {
		return nil, err
	}
	if err := s.ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s Store) ensureSchema(db *sql.DB) error {
	statements := []string{
		`PRAGMA journal_mode=WAL;`,
		`CREATE TABLE IF NOT EXISTS observations (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
			target TEXT NOT NULL,
			campaign TEXT,
			run_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			tags_json TEXT NOT NULL,
			metadata_json TEXT NOT NULL
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
			id UNINDEXED,
			kind,
			title,
			body,
			target,
			campaign,
			tags
		);`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) upsertObservation(db *sql.DB, obs Observation) error {
	tagsJSON, err := json.Marshal(obs.Tags)
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(obs.Metadata)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO observations (id, kind, title, body, target, campaign, run_id, created_at, tags_json, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind,
			title = excluded.title,
			body = excluded.body,
			target = excluded.target,
			campaign = excluded.campaign,
			run_id = excluded.run_id,
			created_at = excluded.created_at,
			tags_json = excluded.tags_json,
			metadata_json = excluded.metadata_json
	`, obs.ID, obs.Kind, obs.Title, obs.Body, obs.Target, obs.Campaign, obs.RunID,
		obs.CreatedAt.UTC().Format(time.RFC3339Nano), string(tagsJSON), string(metadataJSON)); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM observations_fts WHERE id = ?`, obs.ID); err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO observations_fts (id, kind, title, body, target, campaign, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, obs.ID, obs.Kind, obs.Title, obs.Body, obs.Target, obs.Campaign, strings.Join(obs.Tags, " "))
	return err
}

func scanObservations(rows *sql.Rows) ([]Observation, error) {
	items := []Observation{}
	for rows.Next() {
		var (
			item         Observation
			createdAtRaw string
			tagsJSON     string
			metadataJSON string
		)
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Body, &item.Target,
			&item.Campaign, &item.RunID, &createdAtRaw, &tagsJSON, &metadataJSON); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, err
		}
		item.CreatedAt = t
		_ = json.Unmarshal([]byte(tagsJSON), &item.Tags)
		_ = json.Unmarshal([]byte(metadataJSON), &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Store) searchJSONFallback(query string, limit int) ([]Observation, error) {
	items, err := s.loadObservationFiles()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	matched := make([]Observation, 0, len(items))
	for _, item := range items {
		haystack := strings.ToLower(item.Title + "\n" + item.Body + "\n" + item.Target + "\n" + strings.Join(item.Tags, " "))
		if strings.Contains(haystack, q) {
			matched = append(matched, item)
		}
	}
	if limit > 0 && len(matched) > limit {
		return matched[:limit], nil
	}
	return matched, nil
}

func (s Store) recentJSONFallback(limit int) ([]Observation, error) {
	items, err := s.loadObservationFiles()
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(items) > limit {
		return items[:limit], nil
	}
	return items, nil
}

func (s Store) loadObservationFiles() ([]Observation, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, "observations"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	items := make([]Observation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.Root, "observations", entry.Name()))
		if err != nil {
			return nil, err
		}
		var item Observation
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func writeJSON(path string, value interface{}) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func escapeFTSQuery(query string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		term = strings.ReplaceAll(term, `"`, "")
		term = strings.ReplaceAll(term, `'`, "")
		escaped[i] = fmt.Sprintf(`"%s"`, term)
	}
	return strings.Join(escaped, " ")
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}
