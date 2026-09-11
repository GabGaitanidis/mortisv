package userprofile

import (
	"sort"
	"strings"
)


type UserProfile struct {
	Fields map[string][]string `json:"fields"`
}

func NewUserProfile() *UserProfile {
	return &UserProfile{Fields: map[string][]string{}}
}


type FieldUpdate struct {
	Field     string `json:"field"`
	Operation string `json:"operation"` 
	Value     string `json:"value"`
}


func (p *UserProfile) Apply(op FieldUpdate) {
	switch op.Operation {
	case "set":
		p.Fields[op.Field] = []string{op.Value}
	case "add":
		if !contains(p.Fields[op.Field], op.Value) {
			p.Fields[op.Field] = append(p.Fields[op.Field], op.Value)
		}
	case "remove":
		p.Fields[op.Field] = without(p.Fields[op.Field], op.Value)
		if len(p.Fields[op.Field]) == 0 {
			delete(p.Fields, op.Field)
		}
	}
}

func (p *UserProfile) FieldNames() []string {
	names := make([]string, 0, len(p.Fields))
	for k := range p.Fields {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}


func (p *UserProfile) ToPromptString() string {
	names := p.FieldNames()
	if len(names) == 0 {
		return "(no known facts yet)"
	}
	var sb strings.Builder
	for _, field := range names {
		sb.WriteString(field)
		sb.WriteString(": ")
		sb.WriteString(strings.Join(p.Fields[field], ", "))
		sb.WriteString("\n")
	}
	return sb.String()
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func without(list []string, v string) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		if item != v {
			out = append(out, item)
		}
	}
	return out
}
