// Package record is a vertical slice: the model, the persistence layer and the
// HTTP handlers for a single resource. Adding a second resource means adding a
// sibling package, not editing this one.
//
// The resource is deliberately anonymous — a row with a name, a description and
// a flag. It exists to exercise the six HTTP verbs, not to model anything.
package record

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ErrNotFound is returned by the Store when a row does not exist. The handler
// layer maps it to a 404 so storage details never leak into HTTP concerns.
var ErrNotFound = errors.New("record not found")

// Maximum accepted field lengths, enforced in Go and again by the database.
const (
	maxNameLen        = 200
	maxDescriptionLen = 2000
)

// Record is the domain model and, as it happens, the API representation. For a
// bigger service these would be separate types; at this size one is honest.
type Record struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// validateName applies the shared name rules, used by both CREATE and PUT.
func validateName(name string, problems map[string]string) {
	switch {
	case name == "":
		problems["name"] = "name is required"
	case len(name) > maxNameLen:
		problems["name"] = "name must be 200 characters or fewer"
	}
}

// CreateParams is the accepted body for POST.
type CreateParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

// Validate normalises input and reports per-field problems.
func (p *CreateParams) Validate() map[string]string {
	problems := map[string]string{}

	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)

	validateName(p.Name, problems)

	if len(p.Description) > maxDescriptionLen {
		problems["description"] = "description must be 2000 characters or fewer"
	}

	if len(problems) == 0 {
		return nil
	}
	return problems
}

// ReplaceParams is the accepted body for PUT — a *full replacement*.
//
// The fields are plain values, not pointers, and that is the entire point:
// whatever you send is what the row becomes. Omit `description` and it is
// replaced with "". Omit `active` and it is replaced with false. PUT is
// idempotent because the outcome depends only on the body, never on the row's
// current state.
type ReplaceParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

// Validate normalises input and reports per-field problems. Because PUT
// replaces the whole resource, every required field must be present.
func (p *ReplaceParams) Validate() map[string]string {
	problems := map[string]string{}

	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)

	validateName(p.Name, problems)

	if len(p.Description) > maxDescriptionLen {
		problems["description"] = "description must be 2000 characters or fewer"
	}

	if len(problems) == 0 {
		return nil
	}
	return problems
}

// UpdateParams is the accepted body for PATCH — a *partial* update.
//
// Pointer fields let us tell "absent" apart from "set to the zero value". That
// distinction is the whole reason PATCH exists alongside PUT: without it you
// could not send `{"active": false}` and have it mean anything, because it
// would be indistinguishable from `{}`.
type UpdateParams struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Active      *bool   `json:"active"`
}

// Validate normalises input and reports per-field problems. Only the fields
// that were actually supplied are checked.
func (p *UpdateParams) Validate() map[string]string {
	problems := map[string]string{}

	if p.Name != nil {
		trimmed := strings.TrimSpace(*p.Name)
		p.Name = &trimmed

		switch {
		case trimmed == "":
			problems["name"] = "name cannot be empty"
		case len(trimmed) > maxNameLen:
			problems["name"] = "name must be 200 characters or fewer"
		}
	}

	if p.Description != nil {
		trimmed := strings.TrimSpace(*p.Description)
		p.Description = &trimmed

		if len(trimmed) > maxDescriptionLen {
			problems["description"] = "description must be 2000 characters or fewer"
		}
	}

	if p.Name == nil && p.Description == nil && p.Active == nil {
		problems["_"] = "provide at least one of name, description or active"
	}

	if len(problems) == 0 {
		return nil
	}
	return problems
}

// Store is the persistence contract. The handlers depend on this interface
// rather than on Postgres, which is what lets the handler tests run without a
// database.
type Store interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (Record, error)
	Create(ctx context.Context, p CreateParams) (Record, error)
	Replace(ctx context.Context, id int64, p ReplaceParams) (Record, error)
	Update(ctx context.Context, id int64, p UpdateParams) (Record, error)
	Delete(ctx context.Context, id int64) error
}
