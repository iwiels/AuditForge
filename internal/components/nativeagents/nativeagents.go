package nativeagents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orquestador-auditor/internal/agents"
	"orquestador-auditor/internal/assets"
	"orquestador-auditor/internal/components/filemerge"
	"orquestador-auditor/internal/model"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type overlayCapable interface {
	SupportsAgentOverlay() bool
	AgentOverlayPath(homeDir string) string
	EmbeddedAgentOverlayAsset() string
}

type pluginCapable interface {
	SupportsAgentPlugins() bool
	PluginsDir(homeDir string) string
	EmbeddedPluginsDir() string
}

type fileAgentsCapable interface {
	SupportsNativeAgentFiles() bool
	NativeAgentsDir(homeDir string) string
	EmbeddedNativeAgentsDir() string
}

func Inject(homeDir string, adapter agents.Adapter, profile model.AuditProfile, useMarkers bool) error {
	if err := injectOverlay(homeDir, adapter, profile, useMarkers); err != nil {
		return err
	}
	if err := injectPlugins(homeDir, adapter, profile, useMarkers); err != nil {
		return err
	}
	if err := injectFileAgents(homeDir, adapter, profile, useMarkers); err != nil {
		return err
	}
	return nil
}

func ManagedPaths(homeDir string, adapter agents.Adapter, profile model.AuditProfile) []string {
	paths := []string{}
	if a, ok := adapter.(overlayCapable); ok && a.SupportsAgentOverlay() {
		paths = append(paths, a.AgentOverlayPath(homeDir))
	}
	if a, ok := adapter.(pluginCapable); ok && a.SupportsAgentPlugins() && profile.IncludesComponent(model.ComponentSubAgents) {
		entries, _ := assets.List(a.EmbeddedPluginsDir() + "/*")
		for _, entry := range entries {
			paths = append(paths, filepath.Join(a.PluginsDir(homeDir), filepath.Base(entry)))
		}
	}
	if a, ok := adapter.(fileAgentsCapable); ok && a.SupportsNativeAgentFiles() {
		entries, _ := assets.List(a.EmbeddedNativeAgentsDir() + "/*.md")
		for _, entry := range entries {
			name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
			if profile.IncludesComponent(model.ComponentSubAgents) && !profile.AllowsNativeAgent(name) {
				continue
			}
			paths = append(paths, filepath.Join(a.NativeAgentsDir(homeDir), filepath.Base(entry)))
		}
	}
	return paths
}

func AllManagedPaths(homeDir string, adapter agents.Adapter) []string {
	paths := []string{}
	if a, ok := adapter.(overlayCapable); ok && a.SupportsAgentOverlay() {
		paths = append(paths, a.AgentOverlayPath(homeDir))
	}
	if a, ok := adapter.(pluginCapable); ok && a.SupportsAgentPlugins() {
		entries, _ := assets.List(a.EmbeddedPluginsDir() + "/*")
		for _, entry := range entries {
			paths = append(paths, filepath.Join(a.PluginsDir(homeDir), filepath.Base(entry)))
		}
	}
	if a, ok := adapter.(fileAgentsCapable); ok && a.SupportsNativeAgentFiles() {
		entries, _ := assets.List(a.EmbeddedNativeAgentsDir() + "/*.md")
		for _, entry := range entries {
			paths = append(paths, filepath.Join(a.NativeAgentsDir(homeDir), filepath.Base(entry)))
		}
	}
	return paths
}

func injectOverlay(homeDir string, adapter agents.Adapter, profile model.AuditProfile, useMarkers bool) error {
	a, ok := adapter.(overlayCapable)
	if !ok || !a.SupportsAgentOverlay() {
		return nil
	}
	path := a.AgentOverlayPath(homeDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content, err := filteredOverlay(a.EmbeddedAgentOverlayAsset(), profile)
	if err != nil {
		return err
	}
	if useMarkers {
		return injectJSONWithMarkers(path, content)
	}
	return filemerge.WriteJSONMerged(path, content)
}

func injectPlugins(homeDir string, adapter agents.Adapter, profile model.AuditProfile, useMarkers bool) error {
	a, ok := adapter.(pluginCapable)
	if !ok || !a.SupportsAgentPlugins() {
		return nil
	}
	dir := a.PluginsDir(homeDir)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := assets.List(a.EmbeddedPluginsDir() + "/*")
	if err != nil {
		return err
	}
	keep := map[string]struct{}{}
	if profile.IncludesComponent(model.ComponentSubAgents) {
		for _, entry := range entries {
			content, err := assets.Read(entry)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(dir, filepath.Base(entry))
			// Dedicated plugin files must stay valid TypeScript. HTML markers would
			// corrupt the module syntax, so we always write the raw asset instead.
			if err := filemerge.WriteIfChanged(targetPath, sanitizeManagedTextAsset(content)); err != nil {
				return err
			}
			keep[targetPath] = struct{}{}
		}
	}
	for _, entry := range entries {
		targetPath := filepath.Join(dir, filepath.Base(entry))
		if _, ok := keep[targetPath]; ok {
			continue
		}
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func injectFileAgents(homeDir string, adapter agents.Adapter, profile model.AuditProfile, useMarkers bool) error {
	a, ok := adapter.(fileAgentsCapable)
	if !ok || !a.SupportsNativeAgentFiles() {
		return nil
	}
	dir := a.NativeAgentsDir(homeDir)
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := assets.List(a.EmbeddedNativeAgentsDir() + "/*.md")
	if err != nil {
		return err
	}
	keep := map[string]struct{}{}
	for _, entry := range entries {
		name := strings.TrimSuffix(filepath.Base(entry), filepath.Ext(entry))
		targetPath := filepath.Join(dir, filepath.Base(entry))
		if profile.IncludesComponent(model.ComponentSubAgents) && profile.AllowsNativeAgent(name) {
			content, err := assets.Read(entry)
			if err != nil {
				return err
			}
			// OpenCode parses agent frontmatter only when the file starts with `---`.
			// Markers or a UTF-8 BOM before the frontmatter make the agent disappear
			// from the selector, so we always write the raw sanitized markdown file.
			if err := filemerge.WriteIfChanged(targetPath, sanitizeManagedTextAsset(content)); err != nil {
				return err
			}
			keep[targetPath] = struct{}{}
		}
	}
	for _, entry := range entries {
		targetPath := filepath.Join(dir, filepath.Base(entry))
		if _, ok := keep[targetPath]; ok {
			continue
		}
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func filteredOverlay(asset string, profile model.AuditProfile) ([]byte, error) {
	content := assets.MustRead(asset)
	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, err
	}
	agentsMap, _ := payload["agents"].(map[string]any)
	filtered := map[string]any{}
	for name, value := range agentsMap {
		if !profile.IncludesComponent(model.ComponentSubAgents) || !profile.AllowsNativeAgent(name) {
			continue
		}
		agentConfig, ok := value.(map[string]any)
		if !ok {
			continue
		}
		filtered[name] = applyProfilePolicy(agentConfig, profile)
	}
	// Ya no usamos __replace__ en "agent" para no borrar los agentes del usuario.
	// Inyectamos cada agente individualmente en el mapa raíz.
	out := map[string]any{
		"agents": filtered,
	}
	return json.MarshalIndent(out, "", "  ")
}

func applyProfilePolicy(agentConfig map[string]any, profile model.AuditProfile) map[string]any {
	out := map[string]any{}
	for key, value := range agentConfig {
		out[key] = value
	}
	// Remove deprecated "tools" field if present — opencode now uses "permission"
	delete(out, "tools")
	out["mode"] = normalizeAgentMode(stringValue(agentConfig["mode"]))
	out["description"] = fmt.Sprintf("%s | profile=%s | mode=%s", stringValue(agentConfig["description"]), profile.ID, profile.Risk.Mode)
	out["prompt"] = strings.TrimSpace(stringValue(agentConfig["prompt"]) + "\n\n" + profilePromptSuffix(profile))
	out["permission"] = buildPermissionBlock(profile.Risk.Permissions)
	return out
}

// buildPermissionBlock converts ToolPermissions to the opencode "permission" schema.
// Values are "allow" or "deny" strings as required by the current opencode config format.
func buildPermissionBlock(perms model.ToolPermissions) map[string]any {
	permValue := func(enabled bool) string {
		if enabled {
			return "allow"
		}
		return "deny"
	}
	// "edit" is the tool for both writing and editing in opencode.
	// There is no "write" or "webfetch" tool in the official schema.
	editEnabled := perms.Edit || perms.Write

	return map[string]any{
		"read": permValue(perms.Read),
		"edit": permValue(editEnabled),
		"bash": permValue(perms.Bash),
	}
}

func profilePromptSuffix(profile model.AuditProfile) string {
	return fmt.Sprintf("PROFILE POLICY (%s / %s): %s\nAllowed: %s\nBlocked: %s", profile.ID, profile.Risk.Mode, profile.Risk.Summary, strings.Join(profile.Risk.AllowedActions, "; "), strings.Join(profile.Risk.BlockedActions, "; "))
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func normalizeAgentMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "primary", "all", "subagent":
		return strings.TrimSpace(strings.ToLower(mode))
	case "agent":
		return "subagent"
	default:
		return "subagent"
	}
}

// injectJSONWithMarkers writes the overlay to path using JSON-aware merging.
// For JSON files we use filemerge.WriteJSONMerged (which supports __replace__
// sentinels) instead of HTML comment markers that would break JSON parsing.
func injectJSONWithMarkers(path string, overlay []byte) error {
	return filemerge.WriteJSONMerged(path, overlay)
}

func sanitizeManagedTextAsset(content string) []byte {
	return bytes.TrimPrefix([]byte(content), utf8BOM)
}
