package scoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func predicateBands() []ScorePredicate {
	return []ScorePredicate{
		{Name: "D", Min: 0, Max: 59.99},
		{Name: "C", Min: 60, Max: 74.99},
		{Name: "B", Min: 75, Max: 89.99},
		{Name: "A", Min: 90, Max: 100},
	}
}

func TestResolvePredicate(t *testing.T) {
	bands := predicateBands()

	cases := []struct {
		name  string
		score float64
		want  string
	}{
		{"bottom of D band", 0, "D"},
		{"top of D band", 59.99, "D"},
		{"bottom of C band", 60, "C"},
		{"middle of B band", 82, "B"},
		{"bottom of A band", 90, "A"},
		{"top of A band", 100, "A"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolvePredicate(tc.score, bands))
		})
	}

	t.Run("no bands configured returns empty", func(t *testing.T) {
		assert.Equal(t, "", ResolvePredicate(85, nil))
	})

	t.Run("score outside every band returns empty (a school config gap)", func(t *testing.T) {
		gappy := []ScorePredicate{{Name: "A", Min: 90, Max: 100}}
		assert.Equal(t, "", ResolvePredicate(50, gappy))
	})
}

func TestAverage(t *testing.T) {
	t.Run("empty slice is zero, not a division panic", func(t *testing.T) {
		assert.Equal(t, 0.0, Average(nil))
	})

	t.Run("single score", func(t *testing.T) {
		assert.Equal(t, 80.0, Average([]Score{{Score: 80}}))
	})

	t.Run("multiple scores", func(t *testing.T) {
		scores := []Score{{Score: 90}, {Score: 80}, {Score: 70}}
		assert.InDelta(t, 80.0, Average(scores), 0.001)
	})

	t.Run("non-integer average", func(t *testing.T) {
		scores := []Score{{Score: 90}, {Score: 91}}
		assert.InDelta(t, 90.5, Average(scores), 0.001)
	})
}
