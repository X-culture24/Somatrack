// Package resource is a declarative CRUD engine for the school's many simple
// record-keeping forms (visitors book, discipline cases, hostels, expenses…).
//
// Each form is described once as a Resource — table, fields, types, rules,
// who may read and write — and the engine provides validated list/search/
// filter/paginate/get/create/update/delete endpoints plus a JSON form schema
// the portal renders generic forms and tables from. Anything with real
// business logic (fees, payments, marks entry) keeps a hand-written service;
// this is for the long tail.
//
// Safety: every SQL identifier comes from a Resource spec checked at
// registration against a strict pattern; request data only ever travels as
// bind parameters.
package resource

import (
	"fmt"
	"regexp"

	"github.com/stmaryskabete/lms/internal/shared/types"
)

// Type is a field's data type; it drives validation, SQL casts and the
// input widget the portal renders.
type Type string

const (
	Text     Type = "text"
	LongText Type = "longtext"
	Int      Type = "int"
	Decimal  Type = "decimal" // money and weights; 2dp is the caller's concern
	Bool     Type = "bool"
	Date     Type = "date"     // YYYY-MM-DD
	DateTime Type = "datetime" // RFC 3339
	Time     Type = "time"     // HH:MM
	Enum     Type = "enum"
	Ref      Type = "ref" // foreign key (uuid) to another resource
	Phone    Type = "phone"
	Email    Type = "email"
)

// RefSpec describes a foreign key. Label is a SQL expression over alias "x"
// (the referenced table) giving the text shown instead of the raw id, e.g.
// "x.first_name || ' ' || x.last_name".
type RefSpec struct {
	Resource string `json:"resource"` // resource the portal loads options from
	Table    string `json:"-"`
	Label    string `json:"-"`
}

type Field struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     Type     `json:"type"`
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"` // Enum values
	Ref      *RefSpec `json:"ref,omitempty"`
	ReadOnly bool     `json:"read_only,omitempty"` // returned, never written by clients
	Filter   bool     `json:"filterable,omitempty"`
	Search   bool     `json:"searchable,omitempty"` // included in ?search= (text types)
	List     bool     `json:"in_list,omitempty"`    // shown as a table column
	Section  string   `json:"section,omitempty"`    // form grouping heading
	Help     string   `json:"help,omitempty"`
	MaxLen   int      `json:"max_length,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
}

type Resource struct {
	Name   string  `json:"name"`   // URL segment: /api/<name>/
	Title  string  `json:"title"`  // human name
	Module string  `json:"module"` // left-rail module it belongs to
	Fields []Field `json:"fields"`

	Table   string `json:"-"`
	OrderBy string `json:"-"` // SQL over alias t, default "t.created_at DESC"

	// ReadRoles empty = all staff. Parents/students can read only when
	// StudentField is set, and then only rows for their own children.
	ReadRoles  []types.Role `json:"-"`
	WriteRoles []types.Role `json:"-"` // empty = types.AdminRoles
	// StudentField names the column holding the student id, enabling
	// parent/student read access scoped to their own children.
	StudentField string `json:"-"`
	// CreatorField is filled with the caller's user id on create.
	CreatorField string `json:"-"`
	HasUpdatedAt bool   `json:"-"`
	// Constraints maps database constraint names to the message a form
	// shows when that rule is broken.
	Constraints map[string]string `json:"-"`
}

var ident = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)
var urlName = regexp.MustCompile(`^[a-z][a-z0-9-]*(/[a-z][a-z0-9-]*)?$`)

// Check rejects malformed specs. It runs at startup so a typo in a spec
// fails loudly instead of producing a broken endpoint.
func (r *Resource) Check() error {
	if !urlName.MatchString(r.Name) {
		return fmt.Errorf("resource %q: bad name", r.Name)
	}
	if !ident.MatchString(r.Table) {
		return fmt.Errorf("resource %s: bad table %q", r.Name, r.Table)
	}
	for _, c := range []string{r.StudentField, r.CreatorField} {
		if c != "" && !ident.MatchString(c) {
			return fmt.Errorf("resource %s: bad column %q", r.Name, c)
		}
	}
	seen := map[string]bool{}
	for _, f := range r.Fields {
		if !ident.MatchString(f.Name) || f.Name == "id" {
			return fmt.Errorf("resource %s: bad field name %q", r.Name, f.Name)
		}
		if seen[f.Name] {
			return fmt.Errorf("resource %s: duplicate field %q", r.Name, f.Name)
		}
		seen[f.Name] = true
		switch f.Type {
		case Text, LongText, Int, Decimal, Bool, Date, DateTime, Time, Phone, Email:
		case Enum:
			if len(f.Options) == 0 {
				return fmt.Errorf("resource %s: enum %s has no options", r.Name, f.Name)
			}
		case Ref:
			if f.Ref == nil || !ident.MatchString(f.Ref.Table) || f.Ref.Label == "" {
				return fmt.Errorf("resource %s: ref %s needs Table and Label", r.Name, f.Name)
			}
		default:
			return fmt.Errorf("resource %s: field %s has unknown type %q", r.Name, f.Name, f.Type)
		}
		if f.Search && !isText(f.Type) {
			return fmt.Errorf("resource %s: field %s is searchable but not text", r.Name, f.Name)
		}
		if f.Label == "" {
			return fmt.Errorf("resource %s: field %s needs a label", r.Name, f.Name)
		}
	}
	if r.StudentField != "" && !seen[r.StudentField] {
		return fmt.Errorf("resource %s: StudentField %q is not a field", r.Name, r.StudentField)
	}
	return nil
}

func (r *Resource) field(name string) (*Field, bool) {
	for i := range r.Fields {
		if r.Fields[i].Name == name {
			return &r.Fields[i], true
		}
	}
	return nil, false
}

func isText(t Type) bool {
	switch t {
	case Text, LongText, Phone, Email, Enum:
		return true
	}
	return false
}

// Float is a helper for Field.Min/Max literals.
func Float(v float64) *float64 { return &v }
