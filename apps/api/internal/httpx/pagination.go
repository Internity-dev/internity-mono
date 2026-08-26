package httpx

import (
	"strconv"
	"strings"
	"unicode"

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

	return ListParams{Page: page, Limit: limit, Search: sanitizeSearch(c.Query("search")), Sort: sort, Order: order}
}

// sanitizeSearch strips NUL bytes and other non-printable control characters
// out of a raw search query param before it reaches an ILIKE query. Postgres
// rejects NUL bytes in text values outright, and that surfaces as a panic
// (caught by the recovery middleware as an unhandled 500) instead of a clean
// response. Ordinary punctuation, emoji, and other unicode pass through
// untouched — only NUL/control characters are the problem. Returns the
// input unchanged when it's already clean.
func sanitizeSearch(raw string) string {
	if strings.IndexFunc(raw, isUnwantedControl) == -1 {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if !isUnwantedControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isUnwantedControl reports whether r should be stripped from a search
// term: NUL and other C0/C1 control characters, but not the common
// whitespace controls (tab, newline, carriage return) a pasted multi-line
// query might legitimately contain.
func isUnwantedControl(r rune) bool {
	switch r {
	case '\t', '\n', '\r':
		return false
	}
	return unicode.IsControl(r)
}
