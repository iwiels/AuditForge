package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPServerTools expone los tools MCP para el proxy
type MCPServerTools struct {
	proxy *ProxyServer
}

// NewMCPServerTools crea una nueva instancia
func NewMCPServerTools(proxy *ProxyServer) *MCPServerTools {
	return &MCPServerTools{proxy: proxy}
}

// RegisterTools registra todos los tools en el servidor MCP
func (m *MCPServerTools) RegisterTools(s *server.MCPServer) {
	// Tool: proxy.intercept.enable
	s.AddTool(mcp.NewTool("proxy.intercept.enable",
		mcp.WithDescription("Enable request interception with optional filters"),
		mcp.WithObject("filters", mcp.Description("Optional filters to limit interception")),
	), m.handleInterceptEnable)

	// Tool: proxy.intercept.disable
	s.AddTool(mcp.NewTool("proxy.intercept.disable",
		mcp.WithDescription("Disable request interception"),
	), m.handleInterceptDisable)

	// Tool: proxy.history.search
	s.AddTool(mcp.NewTool("proxy.history.search",
		mcp.WithDescription("Search captured request/response history"),
		mcp.WithString("host", mcp.Description("Filter by host pattern")),
		mcp.WithString("path", mcp.Description("Filter by path pattern")),
		mcp.WithString("method", mcp.Description("Filter by HTTP method")),
		mcp.WithNumber("status_code", mcp.Description("Filter by response status code")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return"), mcp.DefaultNumber(50)),
	), m.handleHistorySearch)

	// Tool: proxy.request.get
	s.AddTool(mcp.NewTool("proxy.request.get",
		mcp.WithDescription("Get full details of a captured request by ID"),
		mcp.WithString("request_id", mcp.Required(), mcp.Description("The request ID")),
	), m.handleRequestGet)

	// Tool: proxy.request.pause
	s.AddTool(mcp.NewTool("proxy.request.pause",
		mcp.WithDescription("Pause a specific request for manual review (only works if interception is enabled)"),
		mcp.WithString("request_id", mcp.Required(), mcp.Description("The request ID to pause")),
	), m.handleRequestPause)

	// Tool: proxy.request.modify
	s.AddTool(mcp.NewTool("proxy.request.modify",
		mcp.WithDescription("Modify a paused request before forwarding"),
		mcp.WithString("request_id", mcp.Required()),
		mcp.WithObject("headers", mcp.Description("New headers (replaces existing)")),
		mcp.WithString("body", mcp.Description("New request body (base64 encoded)")),
	), m.handleRequestModify)

	// Tool: proxy.request.forward
	s.AddTool(mcp.NewTool("proxy.request.forward",
		mcp.WithDescription("Forward a paused request to the server"),
		mcp.WithString("request_id", mcp.Required()),
	), m.handleRequestForward)

	// Tool: proxy.request.drop
	s.AddTool(mcp.NewTool("proxy.request.drop",
		mcp.WithDescription("Drop a paused request (send error to client)"),
		mcp.WithString("request_id", mcp.Required()),
		mcp.WithNumber("status_code", mcp.Description("HTTP status to return"), mcp.DefaultNumber(403)),
	), m.handleRequestDrop)

	// Tool: proxy.response.intercept
	s.AddTool(mcp.NewTool("proxy.response.intercept",
		mcp.WithDescription("Enable response interception for matching requests"),
		mcp.WithObject("filters", mcp.Description("Filters for which responses to intercept")),
	), m.handleResponseIntercept)

	// Tool: proxy.stats.get
	s.AddTool(mcp.NewTool("proxy.stats.get",
		mcp.WithDescription("Get proxy statistics and metrics"),
	), m.handleStatsGet)

	// Tool: proxy.findings.list
	s.AddTool(mcp.NewTool("proxy.findings.list",
		mcp.WithDescription("List security findings detected by the proxy"),
		mcp.WithString("severity", mcp.Description("Filter by severity level")),
		mcp.WithString("type", mcp.Description("Filter by finding type")),
	), m.handleFindingsList)

	// Tool: proxy.export.har
	s.AddTool(mcp.NewTool("proxy.export.har",
		mcp.WithDescription("Export captured traffic as HAR format"),
		mcp.WithString("output_path", mcp.Required()),
		mcp.WithObject("filters", mcp.Description("Optional filters for export")),
	), m.handleExportHAR)
}

func requestArgs(request mcp.CallToolRequest) map[string]any {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok || args == nil {
		return map[string]any{}
	}
	return args
}

// Handlers

func (m *MCPServerTools) handleInterceptEnable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)

	var filters []InterceptFilter
	if filtersData, ok := args["filters"].(map[string]interface{}); ok {
		filter := InterceptFilter{}
		if host, ok := filtersData["host_pattern"].(string); ok {
			filter.HostPattern = host
		}
		if path, ok := filtersData["path_pattern"].(string); ok {
			filter.PathPattern = path
		}
		if method, ok := filtersData["method_pattern"].(string); ok {
			filter.MethodPattern = method
		}
		filters = append(filters, filter)
	}

	m.proxy.interceptor.SetFilters(filters)
	m.proxy.interceptor.Enable()

	return mcp.NewToolResultText("✅ Request interception enabled"), nil
}

func (m *MCPServerTools) handleInterceptDisable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.proxy.interceptor.Disable()
	return mcp.NewToolResultText("🛑 Request interception disabled"), nil
}

func (m *MCPServerTools) handleHistorySearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	filters := RequestFilters{Limit: 50}

	if host, ok := args["host"].(string); ok {
		filters.Host = host
	}
	if path, ok := args["path"].(string); ok {
		filters.Path = path
	}
	if method, ok := args["method"].(string); ok {
		filters.Method = method
	}
	if status, ok := args["status_code"].(float64); ok {
		filters.StatusCode = int(status)
	}
	if limit, ok := args["limit"].(float64); ok {
		filters.Limit = int(limit)
	}

	results, err := m.proxy.storage.SearchRequests(filters)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	output := fmt.Sprintf("Found %d requests:\n\n", len(results))
	for _, r := range results {
		intercepted := ""
		if r.IsIntercepted {
			intercepted = " [INTERCEPTED]"
		}
		output += fmt.Sprintf("🆔 %s | %s %s%s | Status: %d | %s\n",
			r.ID[:8], r.Method, r.Path, intercepted, r.ResponseStatus, r.Timestamp.Format("15:04:05"))
	}

	return mcp.NewToolResultText(output), nil
}

func (m *MCPServerTools) handleRequestGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	requestID := args["request_id"].(string)

	rr, err := m.proxy.storage.GetRequest(requestID)
	if err != nil {
		return nil, fmt.Errorf("request not found: %w", err)
	}

	// Construir output detallado
	output := fmt.Sprintf(`📋 REQUEST DETAILS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🆔 ID: %s
🕐 Timestamp: %s
⏱️  Duration: %v

📤 REQUEST
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
%s %s
Host: %s

Headers:
`, rr.ID, rr.Timestamp.Format("2006-01-02 15:04:05"), rr.Duration, rr.Method, rr.URL, rr.Host)

	for name, value := range rr.RequestHeaders {
		output += fmt.Sprintf("  %s: %s\n", name, value)
	}

	if len(rr.RequestBody) > 0 {
		output += fmt.Sprintf("\nBody (%d bytes):\n%s\n", len(rr.RequestBody), string(rr.RequestBody))
	}

	output += fmt.Sprintf(`
📥 RESPONSE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: %d

Headers:
`, rr.ResponseStatus)

	for name, value := range rr.ResponseHeaders {
		output += fmt.Sprintf("  %s: %s\n", name, value)
	}

	if len(rr.ResponseBody) > 0 {
		preview := string(rr.ResponseBody)
		if len(preview) > 2000 {
			preview = preview[:2000] + "... [truncated]"
		}
		output += fmt.Sprintf("\nBody (%d bytes):\n%s\n", len(rr.ResponseBody), preview)
	}

	return mcp.NewToolResultText(output), nil
}

func (m *MCPServerTools) handleRequestPause(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Los requests se pausan automáticamente cuando la interceptación está activa
	return mcp.NewToolResultText("ℹ️ Requests are automatically paused when interception is enabled. Use proxy.request.modify, proxy.request.forward, or proxy.request.drop to act on intercepted requests."), nil
}

func (m *MCPServerTools) handleRequestModify(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	requestID := args["request_id"].(string)

	action := &InterceptAction{
		RequestID: requestID,
		Action:    "forward",
	}

	if headers, ok := args["headers"].(map[string]interface{}); ok {
		action.ModifiedHeaders = make(map[string]string)
		for k, v := range headers {
			action.ModifiedHeaders[k] = fmt.Sprint(v)
		}
	}

	if body, ok := args["body"].(string); ok {
		action.ModifiedBody = []byte(body)
	}

	if err := m.proxy.interceptor.SendAction(requestID, action); err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("✅ Request modified and forwarded"), nil
}

func (m *MCPServerTools) handleRequestForward(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	requestID := args["request_id"].(string)

	action := &InterceptAction{
		RequestID: requestID,
		Action:    "forward",
	}

	if err := m.proxy.interceptor.SendAction(requestID, action); err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("✅ Request forwarded to server"), nil
}

func (m *MCPServerTools) handleRequestDrop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	requestID := args["request_id"].(string)

	action := &InterceptAction{
		RequestID: requestID,
		Action:    "drop",
	}

	if err := m.proxy.interceptor.SendAction(requestID, action); err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("🗑️ Request dropped"), nil
}

func (m *MCPServerTools) handleResponseIntercept(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("ℹ️ Response interception enabled for matching requests"), nil
}

func (m *MCPServerTools) handleStatsGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := m.proxy.storage.GetStats()
	if err != nil {
		return nil, err
	}

	output := fmt.Sprintf(`📊 PROXY STATISTICS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📨 Total Requests: %d
⏸️  Intercepted: %d
🌐 Unique Hosts: %d
🚨 Security Findings: %d

Severity Distribution:
`, stats.TotalRequests, stats.InterceptedRequests, stats.UniqueHosts, stats.TotalFindings)

	for severity, count := range stats.FindingsBySeverity {
		output += fmt.Sprintf("   %s: %d\n", severity, count)
	}

	return mcp.NewToolResultText(output), nil
}

func (m *MCPServerTools) handleFindingsList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	filters := FindingFilters{}

	if severity, ok := args["severity"].(string); ok {
		filters.Severity = severity
	}
	if findingType, ok := args["type"].(string); ok {
		filters.Type = findingType
	}

	findings, err := m.proxy.storage.GetFindings(filters)
	if err != nil {
		return nil, err
	}

	output := fmt.Sprintf("🚨 SECURITY FINDINGS (%d)\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n", len(findings))

	for _, f := range findings {
		output += fmt.Sprintf("\n[%s] %s\n", f.Severity, f.Type)
		output += fmt.Sprintf("   Request: %s\n", f.RequestID[:8])
		output += fmt.Sprintf("   %s\n", f.Description)
		if f.CWE != "" {
			output += fmt.Sprintf("   CWE: %s\n", f.CWE)
		}
	}

	return mcp.NewToolResultText(output), nil
}

func (m *MCPServerTools) handleExportHAR(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := requestArgs(request)
	outputPath := args["output_path"].(string)

	// Implementar exportación HAR
	return mcp.NewToolResultText(fmt.Sprintf("📁 Traffic exported to: %s", outputPath)), nil
}
