package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAverageRating(t *testing.T) {
	t.Run("empty is zero, not a division panic", func(t *testing.T) {
		assert.Equal(t, 0.0, AverageRating(nil))
	})

	t.Run("single review", func(t *testing.T) {
		assert.Equal(t, 4.0, AverageRating([]Review{{Rating: 4}}))
	})

	t.Run("multiple reviews", func(t *testing.T) {
		reviews := []Review{{Rating: 5}, {Rating: 3}, {Rating: 4}}
		assert.InDelta(t, 4.0, AverageRating(reviews), 0.001)
	})
}
