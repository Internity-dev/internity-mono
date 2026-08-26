package internship

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestInternDateEffectiveEndDate(t *testing.T) {
	t.Run("no extension uses end_date", func(t *testing.T) {
		end := date("2026-03-01")
		i := InternDate{EndDate: &end}
		assert.Equal(t, &end, i.EffectiveEndDate())
	})

	t.Run("extension overrides end_date", func(t *testing.T) {
		end := date("2026-03-01")
		extended := date("2026-04-01")
		i := InternDate{EndDate: &end, ExtendedUntil: &extended}
		assert.Equal(t, &extended, i.EffectiveEndDate())
	})

	t.Run("nil end_date and no extension is nil", func(t *testing.T) {
		i := InternDate{}
		assert.Nil(t, i.EffectiveEndDate())
	})
}

func TestInternDateIsActiveOn(t *testing.T) {
	start := date("2026-03-01")
	end := date("2026-03-31")
	i := InternDate{StartDate: &start, EndDate: &end}

	cases := []struct {
		name string
		day  time.Time
		want bool
	}{
		{"before range", date("2026-02-28"), false},
		{"exactly on start", date("2026-03-01"), true},
		{"middle of range", date("2026-03-15"), true},
		{"exactly on end", date("2026-03-31"), true},
		{"after range", date("2026-04-01"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, i.IsActiveOn(tc.day))
		})
	}

	t.Run("unscheduled (nil dates) is never active", func(t *testing.T) {
		unscheduled := InternDate{}
		assert.False(t, unscheduled.IsActiveOn(date("2026-03-15")))
	})

	t.Run("extension pushes the active window out", func(t *testing.T) {
		extended := date("2026-04-15")
		i := InternDate{StartDate: &start, EndDate: &end, ExtendedUntil: &extended}
		assert.True(t, i.IsActiveOn(date("2026-04-10")), "should be active within the extended window")
		assert.False(t, i.IsActiveOn(date("2026-04-20")), "should not be active past the extended window")
	})
}

func TestJournalFilled(t *testing.T) {
	workType := "Development"
	desc := "Worked on the API"
	empty := ""

	cases := []struct {
		name string
		j    Journal
		want bool
	}{
		{"both fields set", Journal{WorkType: &workType, Description: &desc}, true},
		{"nil work_type", Journal{Description: &desc}, false},
		{"nil description", Journal{WorkType: &workType}, false},
		{"both nil", Journal{}, false},
		{"empty string work_type", Journal{WorkType: &empty, Description: &desc}, false},
		{"empty string description", Journal{WorkType: &workType, Description: &empty}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.j.Filled())
		})
	}
}
