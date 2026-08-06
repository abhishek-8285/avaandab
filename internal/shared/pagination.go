package shared

// QueryParams defines standardized parameters for Search, Filter, Sort, and Pagination on all list pages.
type QueryParams struct {
	Search    string `json:"search,omitempty"`
	Status    string `json:"status,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"` // "asc" or "desc"
	Page      int    `json:"page"`
	PerPage   int    `json:"per_page"`
}

// NormalizedPage returns valid page number (min 1).
func (q QueryParams) NormalizedPage() int {
	if q.Page <= 0 {
		return 1
	}
	return q.Page
}

// NormalizedPerPage returns valid items count per page (default 10, max 100).
func (q QueryParams) NormalizedPerPage() int {
	if q.PerPage <= 0 {
		return 10
	}
	if q.PerPage > 100 {
		return 100
	}
	return q.PerPage
}

// Offset returns SQL OFFSET clause value.
func (q QueryParams) Offset() int {
	return (q.NormalizedPage() - 1) * q.NormalizedPerPage()
}

// PagedResult represents a generic Paginated list result wrapper.
type PagedResult[T any] struct {
	Items      []T   `json:"items"`
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

// NewPagedResult constructs a PagedResult with correct TotalPages math.
func NewPagedResult[T any](items []T, total int64, params QueryParams) PagedResult[T] {
	perPage := params.NormalizedPerPage()
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages == 0 {
		totalPages = 1
	}

	return PagedResult[T]{
		Items:      items,
		TotalCount: total,
		Page:       params.NormalizedPage(),
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}
