package httpx

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListParams is the query-param shape every listing endpoint accepts:
// ?page=1&limit=20&search=keyword&sort=created_at&order=desc (+ per-resource filters
// read separately by the caller). One parser for all of them, per spec requirement #12.
type ListParams struct {
	Page   int
	Limit  int
	Search string
	Sort   string
	Order  string
}

func (p ListParams) Offset() int { return (p.Page - 1) * p.Limit }

// ParseListParams reads page/limit/search/sort/order, clamping to sane bounds
// and falling back to defaultSort if the requested sort column isn't in allowedSort.
func ParseListParams(c *gin.Context, defaultSort string, allowedSort map[string]bool) ListParams {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	sort := c.DefaultQuery("sort", defaultSort)
	if !allowedSort[sort] {
		sort = defaultSort
	}
	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return ListParams{Page: page, Limit: limit, Search: c.Query("search"), Sort: sort, Order: order}
}
