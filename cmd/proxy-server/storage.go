package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	_ "github.com/mattn/go-sqlite3"
)

// ProxyStorage maneja la persistencia en SQLite
type ProxyStorage struct {
	db *sql.DB
}

// NewProxyStorage crea una nueva instancia de storage
func NewProxyStorage(dbPath string) (*ProxyStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	storage := &ProxyStorage{db: db}
	if err := storage.createTables(); err != nil {
		return nil, err
	}

	return storage, nil
}

// createTables crea las tablas necesarias
func (s *ProxyStorage) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS requests (
		id TEXT PRIMARY KEY,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		method TEXT,
		url TEXT,
		host TEXT,
		path TEXT,
		query TEXT,
		request_headers TEXT,
		request_body BLOB,
		response_status INTEGER,
		response_headers TEXT,
		response_body BLOB,
		duration_ms INTEGER,
		is_intercepted BOOLEAN DEFAULT 0,
		intercept_action TEXT,
		tags TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_requests_host ON requests(host);
	CREATE INDEX IF NOT EXISTS idx_requests_path ON requests(path);
	CREATE INDEX IF NOT EXISTS idx_requests_method ON requests(method);
	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp);
	CREATE INDEX IF NOT EXISTS idx_requests_intercepted ON requests(is_intercepted);

	CREATE TABLE IF NOT EXISTS findings (
		id TEXT PRIMARY KEY,
		request_id TEXT,
		finding_type TEXT,
		severity TEXT,
		description TEXT,
		evidence TEXT,
		cwe TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (request_id) REFERENCES requests(id)
	);

	CREATE INDEX IF NOT EXISTS idx_findings_type ON findings(finding_type);
	CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveRequest guarda un request/response
func (s *ProxyStorage) SaveRequest(rr *RequestResponse) error {
	reqHeaders, _ := json.Marshal(rr.RequestHeaders)
	respHeaders, _ := json.Marshal(rr.ResponseHeaders)
	tags, _ := json.Marshal(rr.Tags)

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO requests 
		(id, timestamp, method, url, host, path, query, request_headers, request_body,
		 response_status, response_headers, response_body, duration_ms, is_intercepted, intercept_action, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rr.ID, rr.Timestamp, rr.Method, rr.URL, rr.Host, rr.Path, rr.Query,
		reqHeaders, rr.RequestBody, rr.ResponseStatus, respHeaders, rr.ResponseBody,
		int(rr.Duration.Milliseconds()), rr.IsIntercepted, rr.InterceptAction, tags,
	)

	return err
}

// GetRequest obtiene un request por ID
func (s *ProxyStorage) GetRequest(id string) (*RequestResponse, error) {
	var rr RequestResponse
	var reqHeaders, respHeaders, tags string

	err := s.db.QueryRow(`
		SELECT id, timestamp, method, url, host, path, query, request_headers, request_body,
		       response_status, response_headers, response_body, duration_ms, is_intercepted, intercept_action, tags
		FROM requests WHERE id = ?`, id).Scan(
		&rr.ID, &rr.Timestamp, &rr.Method, &rr.URL, &rr.Host, &rr.Path, &rr.Query,
		&reqHeaders, &rr.RequestBody, &rr.ResponseStatus, &respHeaders, &rr.ResponseBody,
		&rr.Duration, &rr.IsIntercepted, &rr.InterceptAction, &tags,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(reqHeaders), &rr.RequestHeaders)
	json.Unmarshal([]byte(respHeaders), &rr.ResponseHeaders)
	json.Unmarshal([]byte(tags), &rr.Tags)

	return &rr, nil
}

// SearchRequests busca requests con filtros
func (s *ProxyStorage) SearchRequests(filters RequestFilters) ([]*RequestResponse, error) {
	query := `SELECT id, timestamp, method, url, host, path, response_status, duration_ms, is_intercepted 
	          FROM requests WHERE 1=1`
	args := []interface{}{}

	if filters.Host != "" {
		query += " AND host LIKE ?"
		args = append(args, "%"+filters.Host+"%")
	}
	if filters.Path != "" {
		query += " AND path LIKE ?"
		args = append(args, "%"+filters.Path+"%")
	}
	if filters.Method != "" {
		query += " AND method = ?"
		args = append(args, filters.Method)
	}
	if filters.StatusCode > 0 {
		query += " AND response_status = ?"
		args = append(args, filters.StatusCode)
	}
	if filters.ContainsBody != nil && *filters.ContainsBody {
		query += " AND LENGTH(request_body) > 0"
	}
	if filters.InterceptedOnly {
		query += " AND is_intercepted = 1"
	}
	if filters.Since != nil {
		query += " AND timestamp > ?"
		args = append(args, filters.Since)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*RequestResponse
	for rows.Next() {
		var rr RequestResponse
		err := rows.Scan(&rr.ID, &rr.Timestamp, &rr.Method, &rr.URL, &rr.Host, &rr.Path,
			&rr.ResponseStatus, &rr.Duration, &rr.IsIntercepted)
		if err != nil {
			continue
		}
		results = append(results, &rr)
	}

	return results, nil
}

// GetInterceptedRequests obtiene requests actualmente interceptados
func (s *ProxyStorage) GetInterceptedRequests() ([]*RequestResponse, error) {
	return s.SearchRequests(RequestFilters{
		InterceptedOnly: true,
		Limit:           100,
	})
}

// DeleteOldRequests elimina requests antiguos
func (s *ProxyStorage) DeleteOldRequests(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.Exec("DELETE FROM requests WHERE timestamp < ?", cutoff)
	return err
}

// SaveFinding guarda un hallazgo de seguridad
func (s *ProxyStorage) SaveFinding(finding *SecurityFinding) error {
	evidence, _ := json.Marshal(finding.Evidence)

	_, err := s.db.Exec(`
		INSERT INTO findings (id, request_id, finding_type, severity, description, evidence, cwe)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		finding.ID, finding.RequestID, finding.Type, finding.Severity,
		finding.Description, evidence, finding.CWE,
	)

	return err
}

// GetFindings obtiene hallazgos filtrados
func (s *ProxyStorage) GetFindings(filters FindingFilters) ([]*SecurityFinding, error) {
	query := `SELECT id, request_id, finding_type, severity, description, evidence, cwe, created_at 
	          FROM findings WHERE 1=1`
	args := []interface{}{}

	if filters.Type != "" {
		query += " AND finding_type = ?"
		args = append(args, filters.Type)
	}
	if filters.Severity != "" {
		query += " AND severity = ?"
		args = append(args, filters.Severity)
	}
	if filters.MinSeverity != "" {
		query += " AND severity IN (SELECT severity FROM severity_order WHERE priority >= (SELECT priority FROM severity_order WHERE severity = ?))"
		args = append(args, filters.MinSeverity)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*SecurityFinding
	for rows.Next() {
		var f SecurityFinding
		var evidence string
		err := rows.Scan(&f.ID, &f.RequestID, &f.Type, &f.Severity, &f.Description, &evidence, &f.CWE, &f.CreatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(evidence), &f.Evidence)
		results = append(results, &f)
	}

	return results, nil
}

// GetStats retorna estadísticas del tráfico
func (s *ProxyStorage) GetStats() (*ProxyStats, error) {
	stats := &ProxyStats{}

	// Total requests
	err := s.db.QueryRow("SELECT COUNT(*) FROM requests").Scan(&stats.TotalRequests)
	if err != nil {
		return nil, err
	}

	// Intercepted requests
	err = s.db.QueryRow("SELECT COUNT(*) FROM requests WHERE is_intercepted = 1").Scan(&stats.InterceptedRequests)
	if err != nil {
		return nil, err
	}

	// Unique hosts
	err = s.db.QueryRow("SELECT COUNT(DISTINCT host) FROM requests").Scan(&stats.UniqueHosts)
	if err != nil {
		return nil, err
	}

	// Total findings
	err = s.db.QueryRow("SELECT COUNT(*) FROM findings").Scan(&stats.TotalFindings)
	if err != nil {
		return nil, err
	}

	// Findings by severity
	rows, err := s.db.Query("SELECT severity, COUNT(*) FROM findings GROUP BY severity")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.FindingsBySeverity = make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		rows.Scan(&severity, &count)
		stats.FindingsBySeverity[severity] = count
	}

	return stats, nil
}

// Close cierra la conexión a la base de datos
func (s *ProxyStorage) Close() error {
	return s.db.Close()
}

// RequestFilters define filtros para búsqueda
type RequestFilters struct {
	Host            string
	Path            string
	Method          string
	StatusCode      int
	ContainsBody    *bool
	InterceptedOnly bool
	Since           *time.Time
	Limit           int
}

// SecurityFinding representa un hallazgo de seguridad
type SecurityFinding struct {
	ID          string                 `json:"id"`
	RequestID   string                 `json:"request_id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Evidence    map[string]interface{} `json:"evidence"`
	CWE         string                 `json:"cwe"`
	CreatedAt   time.Time              `json:"created_at"`
}

// FindingFilters define filtros para hallazgos
type FindingFilters struct {
	Type        string
	Severity    string
	MinSeverity string
}

// ProxyStats contiene estadísticas del proxy
type ProxyStats struct {
	TotalRequests      int            `json:"total_requests"`
	InterceptedRequests int           `json:"intercepted_requests"`
	UniqueHosts        int            `json:"unique_hosts"`
	TotalFindings      int            `json:"total_findings"`
	FindingsBySeverity map[string]int `json:"findings_by_severity"`
}
