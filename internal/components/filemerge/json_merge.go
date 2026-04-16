package filemerge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

const replaceSentinel = "__replace__"

func WriteJSONMerged(path string, overlay []byte) error {
	current, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	merged, err := MergeJSONObjects(current, overlay)
	if err != nil {
		return err
	}
	if bytes.Equal(current, merged) {
		return nil
	}
	return os.WriteFile(path, merged, 0o644)
}

func MergeJSONObjects(baseJSON, overlayJSON []byte) ([]byte, error) {
	base, err := unmarshalJSONObject(baseJSON)
	if err != nil {
		base = map[string]any{}
	}
	overlay, err := unmarshalJSONObject(overlayJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal overlay json: %w", err)
	}
	merged := mergeObjects(base, overlay)
	
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(merged); err != nil {
		return nil, err
	}
	
	return append(buf.Bytes(), '\n'), nil
}

func unmarshalJSONObject(raw []byte) (map[string]any, error) {
	object := map[string]any{}
	if len(bytes.TrimSpace(raw)) == 0 {
		return object, nil
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		return object, nil
	}
	normalized := normalizeJSON(raw)
	if err := json.Unmarshal(normalized, &object); err != nil {
		return nil, err
	}
	return object, nil
}

func normalizeJSON(raw []byte) []byte {
	return stripTrailingCommas(stripJSONComments(raw))
}

func stripJSONComments(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	inString := false
	escaped := false
	inLineComment := false
	inBlockComment := false
	inHTMLComment := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				out = append(out, ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(raw) && raw[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if inHTMLComment {
			if ch == '-' && i+2 < len(raw) && raw[i+1] == '-' && raw[i+2] == '>' {
				inHTMLComment = false
				i += 2
			}
			continue
		}
		if inString {
			out = append(out, ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}
		if ch == '/' && i+1 < len(raw) {
			next := raw[i+1]
			if next == '/' {
				inLineComment = true
				i++
				continue
			}
			if next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		if ch == '<' && i+3 < len(raw) && raw[i+1] == '!' && raw[i+2] == '-' && raw[i+3] == '-' {
			inHTMLComment = true
			i += 3
			continue
		}
		out = append(out, ch)
	}
	return out
}

func stripTrailingCommas(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	inString := false
	escaped := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if inString {
			out = append(out, ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			out = append(out, ch)
			continue
		}
		if ch == ',' {
			j := i + 1
			for j < len(raw) {
				next := raw[j]
				if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
					j++
					continue
				}
				if next == '}' || next == ']' {
					ch = 0
				}
				break
			}
		}
		if ch != 0 {
			out = append(out, ch)
		}
	}
	return out
}

func mergeObjects(base, overlay map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		result[key] = value
	}
	for key, overlayValue := range overlay {
		if replacement, ok := asSentinel(overlayValue); ok {
			result[key] = replacement
			continue
		}
		baseValue, exists := result[key]
		if !exists {
			if overlayMap, isMap := overlayValue.(map[string]any); isMap {
				result[key] = mergeObjects(map[string]any{}, overlayMap)
			} else {
				result[key] = overlayValue
			}
			continue
		}
		baseMap, baseIsMap := baseValue.(map[string]any)
		overlayMap, overlayIsMap := overlayValue.(map[string]any)
		if baseIsMap && overlayIsMap {
			result[key] = mergeObjects(baseMap, overlayMap)
			continue
		}
		result[key] = overlayValue
	}
	return result
}

func asSentinel(v any) (any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	replacement, ok := m[replaceSentinel]
	if !ok || len(m) != 1 {
		return nil, false
	}
	return replacement, true
}
