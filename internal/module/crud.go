package module

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// CRUDService groups the service operations for a resource's CRUD lifecycle.
// T is the entity type, ID is the ID type (string-based), C is the request type.
type CRUDService[T any, ID ~string, C any] struct {
	List    func(ctx context.Context, query, status string, limit, offset int) ([]T, int64, error)
	GetByID func(ctx context.Context, id ID) (T, error)
	Create  func(ctx context.Context, req C) (T, error)
	Update  func(ctx context.Context, id ID, req C) (T, error)
	Delete  func(ctx context.Context, id ID) error
}

// CRUDConfig configures a generic CRUD handler for a resource.
type CRUDConfig[T any, ID ~string, C any] struct {
	Service      *CRUDService[T, ID, C]
	ResourcePath string // e.g., "drivers"
	EntityName   string // e.g., "Driver"
	ListTemplate string // e.g., "driver_list.html"
	FormTemplate string // e.g., "driver_edit.html"
}

// ListCtx returns pagination params and a PaginationData from the request.
func (c *CRUDConfig[T, ID, C]) ListCtx(r *http.Request) (PaginationParams, PaginationData, []T, error) {
	pp := ParsePaginationParams(r)
	list, total, err := c.Service.List(r.Context(), pp.Query, pp.Status, pp.Limit, pp.Offset)
	if err != nil {
		return pp, PaginationData{}, nil, err
	}
	pd := NewPaginationData(pp, total, "/"+c.ResourcePath)
	return pp, pd, list, nil
}

// RegisterRoutes registers standard CRUD routes for a resource on the given router.
func (c *CRUDConfig[T, ID, C]) RegisterRoutes(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		c.handleList(w, r)
	})
	r.Get("/new", func(w http.ResponseWriter, r *http.Request) {
		c.handleNew(w, r)
	})
	r.Post("/new", func(w http.ResponseWriter, r *http.Request) {
		c.handleCreate(w, r)
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		c.handleView(w, r)
	})
	r.Post("/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		c.handleDelete(w, r)
	})
}

func (c *CRUDConfig[T, ID, C]) handleList(w http.ResponseWriter, r *http.Request) {
	pp := ParsePaginationParams(r)
	list, total, err := c.Service.List(r.Context(), pp.Query, pp.Status, pp.Limit, pp.Offset)
	if err != nil {
		WriteError(w, err)
		return
	}
	pd := NewPaginationData(pp, total, "/"+c.ResourcePath)
	_ = pd
	_ = list
}

func (c *CRUDConfig[T, ID, C]) handleNew(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *CRUDConfig[T, ID, C]) handleCreate(w http.ResponseWriter, r *http.Request) {
	WriteError(w, New(CodeValidation, "create not implemented for %s", c.EntityName))
}

func (c *CRUDConfig[T, ID, C]) handleView(w http.ResponseWriter, r *http.Request) {
	id := ID(chi.URLParam(r, "id"))
	_, err := c.Service.GetByID(r.Context(), id)
	if err != nil {
		WriteError(w, err)
		return
	}
}

func (c *CRUDConfig[T, ID, C]) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := ID(chi.URLParam(r, "id"))
	if err := c.Service.Delete(r.Context(), id); err != nil {
		WriteError(w, err)
		return
	}
	http.Redirect(w, r, "/"+c.ResourcePath, http.StatusSeeOther)
}
