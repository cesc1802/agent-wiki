// Package schema loads schema.yaml (the single source of truth for the
// frontmatter contract) and validates wiki page frontmatter against it.
package schema

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"nvtwiki/internal/wiki"
)

// FieldSpec describes one frontmatter field.
type FieldSpec struct {
	Type     string   `yaml:"type"` // string | enum | list | date
	Required bool     `yaml:"required"`
	Nullable bool     `yaml:"nullable"`
	Values   []string `yaml:"values"` // allowed values when Type == "enum"
}

// Schema is the parsed schema.yaml contract.
type Schema struct {
	// Exempt lists page base names that skip validation (e.g. index.md).
	Exempt []string `yaml:"exempt"`
	// Fields maps a frontmatter key to its specification.
	Fields map[string]FieldSpec `yaml:"fields"`
}

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Load reads and parses schema.yaml from path.
func Load(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse schema %s: %w", path, err)
	}
	if len(s.Fields) == 0 {
		return nil, fmt.Errorf("schema %s declares no fields", path)
	}
	return &s, nil
}

func (s *Schema) isExempt(base string) bool {
	for _, e := range s.Exempt {
		if e == base {
			return true
		}
	}
	return false
}

// ValidatePage returns the list of validation problems for one page. An exempt
// page returns no problems.
func (s *Schema) ValidatePage(p wiki.Page) []string {
	if s.isExempt(p.Base()) {
		return nil
	}
	if !p.HasFront {
		return []string{"missing frontmatter block"}
	}
	if p.FrontErr != nil {
		return []string{fmt.Sprintf("invalid frontmatter YAML: %v", p.FrontErr)}
	}

	var problems []string
	// Validate declared fields in a stable order for deterministic output.
	names := make([]string, 0, len(s.Fields))
	for name := range s.Fields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		spec := s.Fields[name]
		val, present := p.Front[name]
		if !present {
			if spec.Required {
				problems = append(problems, fmt.Sprintf("missing required field %q", name))
			}
			continue
		}
		if val == nil {
			if !spec.Nullable {
				problems = append(problems, fmt.Sprintf("field %q must not be null", name))
			}
			continue
		}
		if msg := checkType(name, spec, val); msg != "" {
			problems = append(problems, msg)
		}
	}
	return problems
}

func checkType(name string, spec FieldSpec, val interface{}) string {
	switch spec.Type {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Sprintf("field %q must be a string", name)
		}
	case "date":
		// YAML decodes an unquoted YYYY-MM-DD into time.Time; a quoted value
		// stays a string. Accept both, but require the date-only shape.
		switch v := val.(type) {
		case time.Time:
			// A parsed timestamp is a valid date.
		case string:
			if !dateRe.MatchString(v) {
				return fmt.Sprintf("field %q must match YYYY-MM-DD, got %q", name, v)
			}
		default:
			return fmt.Sprintf("field %q must be a date (YYYY-MM-DD)", name)
		}
	case "enum":
		str, ok := val.(string)
		if !ok {
			return fmt.Sprintf("field %q must be one of [%s]", name, strings.Join(spec.Values, ", "))
		}
		for _, allowed := range spec.Values {
			if str == allowed {
				return ""
			}
		}
		return fmt.Sprintf("field %q value %q not in [%s]", name, str, strings.Join(spec.Values, ", "))
	case "list":
		if _, ok := val.([]interface{}); !ok {
			return fmt.Sprintf("field %q must be a list", name)
		}
	default:
		return fmt.Sprintf("field %q has unknown schema type %q", name, spec.Type)
	}
	return ""
}
