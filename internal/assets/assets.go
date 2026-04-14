package assets

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed prompts/*.md
//go:embed opencode/commands/*.md opencode/agents/*.md opencode/plugins/*.ts
//go:embed claude/*.md claude/commands/*.md
//go:embed cursor/agents/*.md
//go:embed gemini/GEMINI.md gemini/commands/*.toml gemini/agents/*.md
//go:embed skills/*/*.md
//go:embed wordlists/*.txt
var embedded embed.FS

func Read(name string) (string, error) {
	b, err := embedded.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func MustRead(name string) string {
	content, err := Read(name)
	if err != nil {
		panic(err)
	}
	return content
}

func List(pattern string) ([]string, error) {
	matches, err := fs.Glob(embedded, pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}
