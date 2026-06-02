package handlers

import (
	"net/http"
	"strconv"
)

// PaginationParams holds parsed pagination query parameters.
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Offset returns the SQL offset for the current page.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// PaginationResponse holds pagination metadata for API responses.
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// ParsePagination extracts page and page_size from query params with defaults and limits.
func ParsePagination(r *http.Request) PaginationParams {
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return PaginationParams{Page: page, PageSize: pageSize}
}

// NewPaginationResponse builds a PaginationResponse from total count and params.
func NewPaginationResponse(totalItems int, params PaginationParams) PaginationResponse {
	totalPages := 0
	if params.PageSize > 0 {
		totalPages = (totalItems + params.PageSize - 1) / params.PageSize
	}
	return PaginationResponse{
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
