package zbz

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

// Macro defines the interface for executing stored SQL queries with parameters
type Macro interface {
	Interpolate(embed map[string]string) (string, error)
}

// MacroContract defines the necessary data to implement a macro as a query
type MacroContract struct {
	Name  string
	Macro string
	Embed map[string]string
}

// zMacro represents a SQL query with its metadata
type zMacro struct {
	Name       string
	Template   string
	Embeddings []string
}

// NewMacro creates a new Macro from a given SQLx template
func NewMacro(name string, template string) Macro {
	embeddings := []string{}
	lines := strings.Split(template, "\n")
	t := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "-- @embed") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				embeddings = append(embeddings, parts[2])
			}
			continue
		}
		if !strings.HasPrefix(line, "--") && strings.TrimSpace(line) != "" {
			t += line + "\n"
		}
	}
	return &zMacro{
		Name:       name,
		Template:   t,
		Embeddings: embeddings,
	}
}

// Interpolate a query template by replacing placeholders with actual values
func (q *zMacro) Interpolate(embed map[string]string) (string, error) {
	Log.Debugf("Interpolating %s query %v", q.Name, spew.Sdump(embed))
	// TODO embedded content is raw sql - add some sanitization or validation to mitigate SQL injection risk
	query := q.Template
	for _, property := range q.Embeddings {
		if value, ok := embed[property]; ok {
			placeholder := fmt.Sprintf("{{%s}}", property)
			if !strings.Contains(query, placeholder) {
				return "", fmt.Errorf("placeholder %s not found in query template", placeholder)
			}
			query = strings.ReplaceAll(query, placeholder, value)
		} else {
			return "", fmt.Errorf("missing embedding value for `%s`", property)
		}
	}

	Log.Debugf("Interpolated query: %s", query)
	return query, nil
}
