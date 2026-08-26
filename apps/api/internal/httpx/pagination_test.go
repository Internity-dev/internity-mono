package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newListParamsContext(rawQuery string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	return c
}

func TestParseListParams_StripsNULByteFromSearch(t *testing.T) {
	c := newListParamsContext("search=foo%00bar")
	params := ParseListParams(c, "name", map[string]bool{"name": true})
	assert.Equal(t, "foobar", params.Search)
}

func TestParseListParams_StripsOtherControlCharsFromSearch(t *testing.T) {
	c := newListParamsContext("search=foo%01%1Fbar")
	params := ParseListParams(c, "name", map[string]bool{"name": true})
	assert.Equal(t, "foobar", params.Search)
}

func TestParseListParams_LeavesCleanSearchUntouched(t *testing.T) {
	c := newListParamsContext("search=" + `caf%C3%A9%20%F0%9F%9A%80%20%2850%25%20off%29`)
	params := ParseListParams(c, "name", map[string]bool{"name": true})
	assert.Equal(t, "café 🚀 (50% off)", params.Search)
}

func TestParseListParams_KeepsCommonWhitespaceInSearch(t *testing.T) {
	c := newListParamsContext("search=foo%0Abar")
	params := ParseListParams(c, "name", map[string]bool{"name": true})
	assert.Equal(t, "foo\nbar", params.Search)
}

func TestSanitizeSearch_ReturnsSameStringWhenClean(t *testing.T) {
	assert.Equal(t, "already clean", sanitizeSearch("already clean"))
}
