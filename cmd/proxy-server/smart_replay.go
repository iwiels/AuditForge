package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SmartReplayEngine es el motor de análisis diferencial
type SmartReplayEngine struct {
	storage      *ProxyStorage
	detectors    []DetectionRule
	mutationGen  *MutationGenerator
	client       *http.Client
}

// DetectionRule define una regla de detección
type DetectionRule struct {
	Name        string
	Description string
	Type        string
	Severity    string
	CWE         string
	Condition   func(baseline *ReplayResult, variation *ReplayResult) *DetectionFinding
}

// ReplayResult representa el resultado de un replay
type ReplayResult struct {
	RequestID        string
	OriginalRequest  *RequestResponse
	ModifiedRequest  *http.Request
	Response         *http.Response
	ResponseBody     []byte
	ResponseTime     time.Duration
	Timestamp        time.Time
	AppliedMutation  *Mutation
}

// DetectionFinding representa un hallazgo detectado
type DetectionFinding struct {
	Type            string                 `json:"type"`
	Severity        string                 `json:"severity"`
	Description     string                 `json:"description"`
	CWE             string                 `json:"cwe"`
	Evidence        map[string]interface{} `json:"evidence"`
	Recommendation  string                 `json:"recommendation"`
}

// ReplayConfig configura una sesión de replay
type ReplayConfig struct {
	BaseRequestID   string
	Mutations       []Mutation
	Parallelism     int
	DelayBetween    time.Duration
	FollowRedirects bool
}

// Mutation representa una modificación a aplicar
type Mutation struct {
	Name        string
	Type        string // "param", "header", "body", "method", "path"
	Target      string // nombre del campo a modificar
	Operation   string // "replace", "remove", "add", "increment", "fuzz"
	Value       interface{}
	Description string
}

// MutationGenerator genera mutaciones automáticas
type MutationGenerator struct {
	idMutations     []string
	roleMutations   []string
	pathMutations   []string
	headerMutations map[string][]string
}

// NewSmartReplayEngine crea un nuevo motor de replay
func NewSmartReplayEngine(storage *ProxyStorage) *SmartReplayEngine {
	engine := &SmartReplayEngine{
		storage: storage,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // No seguir redirects automáticamente
			},
		},
	}

	engine.mutationGen = NewMutationGenerator()
	engine.registerDefaultDetectors()

	return engine
}

// registerDefaultDetectors registra detectores por defecto
func (e *SmartReplayEngine) registerDefaultDetectors() {
	e.detectors = []DetectionRule{
		{
			Name:        "Status Code Bypass",
			Description: "Detecta cambios de status code que indican bypass de autorización",
			Type:        "auth_bypass",
			Severity:    "CRITICAL",
			CWE:         "CWE-287",
			Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
				if baseline.Response.StatusCode >= 400 && variation.Response.StatusCode < 300 {
					return &DetectionFinding{
						Type:        "auth_bypass_status_change",
						Severity:    "CRITICAL",
						Description: fmt.Sprintf("Authorization bypass: %d → %d", baseline.Response.StatusCode, variation.Response.StatusCode),
						CWE:         "CWE-287",
						Evidence: map[string]interface{}{
							"baseline_status":   baseline.Response.StatusCode,
							"variation_status":  variation.Response.StatusCode,
							"mutation_applied":  variation.AppliedMutation.Description,
						},
						Recommendation: "Verificar que todos los endpoints validen correctamente la autorización",
					}
				}
				return nil
			},
		},
		{
			Name:        "IDOR - Data Access",
			Description: "Detecta acceso a datos de otros usuarios mediante modificación de IDs",
			Type:        "idor",
			Severity:    "HIGH",
			CWE:         "CWE-639",
			Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
				// Si baseline es 404 pero variation devuelve 200 con datos
				if baseline.Response.StatusCode == 404 && variation.Response.StatusCode == 200 {
					bodySize := len(variation.ResponseBody)
					if bodySize > 100 { // Tiene contenido sustancial
						return &DetectionFinding{
							Type:        "idor_data_access",
							Severity:    "HIGH",
							Description: "Posible IDOR: acceso a recursos con ID modificado",
							CWE:         "CWE-639",
							Evidence: map[string]interface{}{
								"accessible_id":    variation.AppliedMutation.Value,
								"response_size":    bodySize,
								"mutation":         variation.AppliedMutation.Name,
							},
							Recommendation: "Implementar verificación de ownership en todos los endpoints que acceden a recursos por ID",
						}
					}
				}
				return nil
			},
		},
		{
			Name:        "Response Schema Leak",
			Description: "Detecta campos adicionales en respuestas que pueden indicar privilege escalation",
			Type:        "schema_leak",
			Severity:    "MEDIUM",
			CWE:         "CWE-200",
			Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
				// Comparar campos en JSON
				var baselineFields, variationFields map[string]interface{}
				json.Unmarshal(baseline.ResponseBody, &baselineFields)
				json.Unmarshal(variation.ResponseBody, &variationFields)

				extraFields := []string{}
				for key := range variationFields {
					if _, exists := baselineFields[key]; !exists {
						extraFields = append(extraFields, key)
					}
				}

				if len(extraFields) > 0 {
					return &DetectionFinding{
						Type:        "schema_field_leak",
						Severity:    "MEDIUM",
						Description: fmt.Sprintf("Campos adicionales en respuesta: %v", extraFields),
						CWE:         "CWE-200",
						Evidence: map[string]interface{}{
							"extra_fields":    extraFields,
							"mutation":        variation.AppliedMutation.Name,
						},
						Recommendation: "Asegurar que la API filtre campos sensibles según el rol del usuario",
					}
				}
				return nil
			},
		},
		{
			Name:        "Timing Side Channel",
			Description: "Detecta diferencias de tiempo que pueden indicar existencia de recursos",
			Type:        "timing_attack",
			Severity:    "LOW",
			CWE:         "CWE-208",
			Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
				// Si hay una diferencia significativa de tiempo
				timeDiff := variation.ResponseTime - baseline.ResponseTime
				if timeDiff > 500*time.Millisecond {
					return &DetectionFinding{
						Type:        "timing_side_channel",
						Severity:    "LOW",
						Description: fmt.Sprintf("Diferencia de tiempo significativa: %v", timeDiff),
						CWE:         "CWE-208",
						Evidence: map[string]interface{}{
							"baseline_time":   baseline.ResponseTime.String(),
							"variation_time":  variation.ResponseTime.String(),
							"difference":      timeDiff.String(),
						},
						Recommendation: "Implementar tiempo constante para operaciones sensibles",
					}
				}
				return nil
			},
		},
		{
			Name:        "Error Message Information Disclosure",
			Description: "Detecta mensajes de error que revelan información interna",
			Type:        "info_disclosure",
			Severity:    "MEDIUM",
			CWE:         "CWE-209",
			Condition: func(baseline, variation *ReplayResult) *DetectionFinding {
				responseStr := string(variation.ResponseBody)
				indicators := []string{
					"stack trace", "exception", "syntax error", "sql syntax",
					"database error", "permission denied", "file not found",
					"/var/www", "/home/", "c:/", "server at",
				}

				found := []string{}
				lowerResponse := strings.ToLower(responseStr)
				for _, indicator := range indicators {
					if strings.Contains(lowerResponse, indicator) {
						found = append(found, indicator)
					}
				}

				if len(found) > 0 {
					return &DetectionFinding{
						Type:        "error_info_disclosure",
						Severity:    "MEDIUM",
						Description: fmt.Sprintf("Información sensible en mensaje de error: %v", found),
						CWE:         "CWE-209",
						Evidence: map[string]interface{}{
							"indicators":  found,
							"status_code": variation.Response.StatusCode,
						},
						Recommendation: "Implementar mensajes de error genéricos para usuarios y loguear detalles internamente",
					}
				}
				return nil
			},
		},
	}
}

// ExecuteBaseline captura la respuesta baseline
func (e *SmartReplayEngine) ExecuteBaseline(requestID string) (*ReplayResult, error) {
	rr, err := e.storage.GetRequest(requestID)
	if err != nil {
		return nil, err
	}

	return e.executeRequest(rr, nil)
}

// ExecuteVariations ejecuta múltiples variaciones
func (e *SmartReplayEngine) ExecuteVariations(baseRequestID string, mutations []Mutation) ([]*ReplayResult, []*DetectionFinding, error) {
	// Obtener baseline
	baseline, err := e.ExecuteBaseline(baseRequestID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	results := []*ReplayResult{baseline}
	var findings []*DetectionFinding

	// Ejecutar cada mutación
	for _, mutation := range mutations {
		result, err := e.executeWithMutation(baseline.OriginalRequest, &mutation)
		if err != nil {
			continue
		}
		results = append(results, result)

		// Analizar diferencias
		for _, detector := range e.detectors {
			if finding := detector.Condition(baseline, result); finding != nil {
				finding.Type = detector.Type
				finding.Severity = detector.Severity
				finding.CWE = detector.CWE
				findings = append(findings, finding)

				// Guardar en storage
				e.storage.SaveFinding(&SecurityFinding{
					ID:          uuid.New().String(),
					RequestID:   baseRequestID,
					Type:        finding.Type,
					Severity:    finding.Severity,
					Description: finding.Description,
					Evidence:    finding.Evidence,
					CWE:         finding.CWE,
				})
			}
		}
	}

	return results, findings, nil
}

// executeRequest ejecuta un request
func (e *SmartReplayEngine) executeRequest(rr *RequestResponse, mutation *Mutation) (*ReplayResult, error) {
	// Construir request
	reqURL, _ := url.Parse(rr.URL)
	req, _ := http.NewRequest(rr.Method, rr.URL, bytes.NewReader(rr.RequestBody))

	// Copiar headers
	for name, value := range rr.RequestHeaders {
		// Skip hop-by-hop headers
		if name == "Proxy-Connection" || name == "Proxy-Authorization" {
			continue
		}
		req.Header.Set(name, value)
	}

	// Aplicar mutación si existe
	if mutation != nil {
		e.applyMutation(req, mutation)
	}

	// Ejecutar
	start := time.Now()
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return &ReplayResult{
		RequestID:       rr.ID,
		OriginalRequest: rr,
		ModifiedRequest: req,
		Response:        resp,
		ResponseBody:    body,
		ResponseTime:    time.Since(start),
		Timestamp:       time.Now(),
		AppliedMutation: mutation,
	}, nil
}

// executeWithMutation ejecuta con una mutación específica
func (e *SmartReplayEngine) executeWithMutation(rr *RequestResponse, mutation *Mutation) (*ReplayResult, error) {
	return e.executeRequest(rr, mutation)
}

// applyMutation aplica una mutación al request
func (e *SmartReplayEngine) applyMutation(req *http.Request, mutation *Mutation) {
	switch mutation.Type {
	case "header":
		switch mutation.Operation {
		case "replace":
			req.Header.Set(mutation.Target, fmt.Sprint(mutation.Value))
		case "remove":
			req.Header.Del(mutation.Target)
		case "add":
			req.Header.Add(mutation.Target, fmt.Sprint(mutation.Value))
		}

	case "param":
		query := req.URL.Query()
		switch mutation.Operation {
		case "replace":
			query.Set(mutation.Target, fmt.Sprint(mutation.Value))
		case "remove":
			query.Del(mutation.Target)
		}
		req.URL.RawQuery = query.Encode()

	case "body":
		if mutation.Operation == "replace" {
			req.Body = io.NopCloser(strings.NewReader(fmt.Sprint(mutation.Value)))
			req.ContentLength = int64(len(fmt.Sprint(mutation.Value)))
		}

	case "method":
		req.Method = fmt.Sprint(mutation.Value)

	case "path":
		req.URL.Path = fmt.Sprint(mutation.Value)
	}
}

// GenerateAutoMutations genera mutaciones automáticas basadas en el request
func (e *SmartReplayEngine) GenerateAutoMutations(rr *RequestResponse) []Mutation {
	mutations := []Mutation{}

	// 1. Mutaciones de ID (IDOR testing)
	mutations = append(mutations, e.mutationGen.GenerateIDMutations(rr)...)

	// 2. Mutaciones de rol/permisos
	mutations = append(mutations, e.mutationGen.GenerateRoleMutations(rr)...)

	// 3. Mutaciones de autenticación
	mutations = append(mutations, e.mutationGen.GenerateAuthMutations(rr)...)

	// 4. Mutaciones de path
	mutations = append(mutations, e.mutationGen.GeneratePathMutations(rr)...)

	return mutations
}

// NewMutationGenerator crea un nuevo generador de mutaciones
func NewMutationGenerator() *MutationGenerator {
	return &MutationGenerator{
		idMutations:   []string{"1", "2", "999", "9999", "0", "-1", "../", "..%2f"},
		roleMutations: []string{"admin", "superadmin", "moderator", "user", "guest"},
		pathMutations: []string{"/admin", "/api/admin", "/internal", "/debug"},
		headerMutations: map[string][]string{
			"X-User-Role":     {"admin", "superadmin"},
			"X-Forwarded-For": {"127.0.0.1", "10.0.0.1", "192.168.1.1"},
			"X-Original-URL":  {"/admin", "/api/internal"},
		},
	}
}

// GenerateIDMutations genera mutaciones para IDs
func (g *MutationGenerator) GenerateIDMutations(rr *RequestResponse) []Mutation {
	mutations := []Mutation{}

	// Buscar IDs en query params
	query := ""
	if u, err := url.Parse(rr.URL); err == nil {
		query = u.RawQuery
	}

	// Patrones comunes de ID
	idPatterns := []string{"id", "user_id", "account_id", "order_id", "product_id"}

	for _, pattern := range idPatterns {
		if strings.Contains(query, pattern) || strings.Contains(rr.Path, pattern) {
			for _, idValue := range g.idMutations {
				mutations = append(mutations, Mutation{
					Name:        fmt.Sprintf("id_fuzz_%s_%s", pattern, idValue),
					Type:        "param",
					Target:      pattern,
					Operation:   "replace",
					Value:       idValue,
					Description: fmt.Sprintf("IDOR test: cambiar %s a %s", pattern, idValue),
				})
			}
		}
	}

	return mutations
}

// GenerateRoleMutations genera mutaciones de rol
func (g *MutationGenerator) GenerateRoleMutations(rr *RequestResponse) []Mutation {
	mutations := []Mutation{}

	for header, roles := range g.headerMutations {
		for _, role := range roles {
			mutations = append(mutations, Mutation{
				Name:        fmt.Sprintf("role_header_%s_%s", header, role),
				Type:        "header",
				Target:      header,
				Operation:   "add",
				Value:       role,
				Description: fmt.Sprintf("Privilege escalation test: agregar header %s=%s", header, role),
			})
		}
	}

	return mutations
}

// GenerateAuthMutations genera mutaciones de autenticación
func (g *MutationGenerator) GenerateAuthMutations(rr *RequestResponse) []Mutation {
	mutations := []Mutation{}

	// Remover Authorization
	if _, hasAuth := rr.RequestHeaders["Authorization"]; hasAuth {
		mutations = append(mutations, Mutation{
			Name:        "auth_remove_bearer",
			Type:        "header",
			Target:      "Authorization",
			Operation:   "remove",
			Value:       nil,
			Description: "Auth bypass test: remover header Authorization",
		})
	}

	// Remover Cookie de sesión
	if _, hasCookie := rr.RequestHeaders["Cookie"]; hasCookie {
		mutations = append(mutations, Mutation{
			Name:        "auth_remove_cookie",
			Type:        "header",
			Target:      "Cookie",
			Operation:   "remove",
			Value:       nil,
			Description: "Auth bypass test: remover cookies",
		})
	}

	return mutations
}

// GeneratePathMutations genera mutaciones de path
func (g *MutationGenerator) GeneratePathMutations(rr *RequestResponse) []Mutation {
	mutations := []Mutation{}

	// Probar paths de admin en APIs que no son de admin
	if !strings.Contains(rr.Path, "admin") {
		for _, adminPath := range g.pathMutations {
			mutations = append(mutations, Mutation{
				Name:        fmt.Sprintf("path_traversal_%s", adminPath),
				Type:        "path",
				Target:      "path",
				Operation:   "replace",
				Value:       adminPath,
				Description: fmt.Sprintf("Path bypass test: cambiar path a %s", adminPath),
			})
		}
	}

	return mutations
}
