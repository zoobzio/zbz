package zbz

import (
	"fmt"
	"strings"

	"zbz/shared/logger"
)

// Macro defines the interface for executing stored SQL queries with parameters
type Macro interface {
	Interpolate(embed MacroEmbeds) (string, error)
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

// Interpolate a query template by replacing placeholders with validated values
func (q *zMacro) Interpolate(embed MacroEmbeds) (string, error) {
	logger.Log.Debug("Interpolating query with trusted values", logger.Any("query", q))

	query := q.Template
	
	// Map of known embeddings to their values from the MacroEmbeds struct
	embedMap := map[string]string{
		"table":   embed.Table.String(),
		"columns": embed.Columns.String(),
		"values":  embed.Values.String(),
		"updates": embed.Updates.String(),
	}

	for _, property := range q.Embeddings {
		if value, ok := embedMap[property]; ok {
			placeholder := fmt.Sprintf("{{%s}}", property)
			if !strings.Contains(query, placeholder) {
				return "", fmt.Errorf("placeholder %s not found in query template", placeholder)
			}
			// Safe to interpolate - value is validated through TrustedSQLIdentifier
			query = strings.ReplaceAll(query, placeholder, value)
		} else {
			return "", fmt.Errorf("missing trusted embedding value for `%s`", property)
		}
	}

	return query, nil
}

// BuildMacroEmbeds creates all embeddings from meta columns with schema validation
func BuildMacroEmbeds(schema Schema, meta *Meta) (MacroEmbeds, error) {
	tableName := strings.ToLower(meta.Name)
	
	// Validate table exists in schema
	if !schema.IsValidTable(tableName) {
		return MacroEmbeds{}, fmt.Errorf("table not found in schema: %s", tableName)
	}
	
	// Build table identifier
	table := TrustedSQLIdentifier{value: tableName}
	
	// Build columns: "id, name, email"
	columnList := strings.Join(meta.ColumnNames, ", ")
	if !schema.IsValidColumns(tableName, meta.ColumnNames) {
		return MacroEmbeds{}, fmt.Errorf("invalid columns for table %s: %s", tableName, columnList)
	}
	columns := TrustedSQLIdentifier{value: columnList}
	
	// Build values: ":id, :name, :email" (always safe for prepared statements)
	values := make([]string, len(meta.ColumnNames))
	for i, col := range meta.ColumnNames {
		values[i] = ":" + col
	}
	valueList := TrustedSQLIdentifier{value: strings.Join(values, ", ")}
	
	// Build updates: "name = :name, email = :email" (skip id)
	updates := make([]string, 0, len(meta.ColumnNames)-1)
	for _, col := range meta.ColumnNames {
		if col != "id" {
			updates = append(updates, fmt.Sprintf("%s = :%s", col, col))
		}
	}
	updateList := TrustedSQLIdentifier{value: strings.Join(updates, ", ")}
	
	return MacroEmbeds{
		Table:   table,
		Columns: columns,
		Values:  valueList,
		Updates: updateList,
	}, nil
}
