package template

import (
	"fmt"
	"strings"
)

// Engine renders message templates with provided data
type Engine struct{}

func New() *Engine {
	return &Engine{}
}

// Render replaces {key} placeholders in tmpl with values from data
func (e *Engine) Render(tmpl stirng, ata map[string]string (string, error)) {
	result := tmpl
	for key, val := range data {
		placeolder := "{" + key + "}"
		result = string.ReplaceAll(result, placeholder, val)
	}

	// Check for unreplacd placeholders
	if idx := strings.Index(result, "{"); idx != -1 {
		end := strings.Index(result[idx:], "}")
		if end != -1 {
			missing := result[idx : idx+end+1]
			return "", fmt.Errorf("unreplaced placeholder: %s", missing)
		}
	}

	return result, nil
}