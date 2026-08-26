package vacancy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplianceCanTransitionTo(t *testing.T) {
	cases := []struct {
		name    string
		from    ApplianceStatus
		to      ApplianceStatus
		allowed bool
	}{
		{"pending -> processed", StatusPending, StatusProcessed, true},
		{"pending -> accepted", StatusPending, StatusAccepted, true},
		{"pending -> rejected", StatusPending, StatusRejected, true},
		{"pending -> canceled", StatusPending, StatusCanceled, true},
		{"processed -> accepted", StatusProcessed, StatusAccepted, true},
		{"processed -> rejected", StatusProcessed, StatusRejected, true},
		{"processed -> canceled", StatusProcessed, StatusCanceled, true},
		{"processed -> pending is not allowed (no going backwards)", StatusProcessed, StatusPending, false},
		{"accepted is terminal: -> rejected", StatusAccepted, StatusRejected, false},
		{"accepted is terminal: -> canceled", StatusAccepted, StatusCanceled, false},
		{"accepted is terminal: -> processed", StatusAccepted, StatusProcessed, false},
		{"rejected is terminal: -> pending", StatusRejected, StatusPending, false},
		{"rejected is terminal: -> accepted", StatusRejected, StatusAccepted, false},
		{"canceled is terminal: -> pending", StatusCanceled, StatusPending, false},
		{"canceled is terminal: -> accepted", StatusCanceled, StatusAccepted, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := Appliance{Status: tc.from}
			err := a.CanTransitionTo(tc.to)
			if tc.allowed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				var transErr ErrInvalidTransition
				assert.ErrorAs(t, err, &transErr)
				assert.Equal(t, tc.from, transErr.From)
				assert.Equal(t, tc.to, transErr.To)
			}
		})
	}
}

func TestApplianceIsTerminal(t *testing.T) {
	assert.False(t, Appliance{Status: StatusPending}.IsTerminal())
	assert.False(t, Appliance{Status: StatusProcessed}.IsTerminal())
	assert.True(t, Appliance{Status: StatusAccepted}.IsTerminal())
	assert.True(t, Appliance{Status: StatusRejected}.IsTerminal())
	assert.True(t, Appliance{Status: StatusCanceled}.IsTerminal())
}

// TestNoTransitionEscapesATerminalState is a completeness sweep — for every
// terminal status, every possible target must be rejected, not just the
// handful spot-checked above. If a future edit accidentally reopens a
// terminal state, this catches it regardless of which target status they add.
func TestNoTransitionEscapesATerminalState(t *testing.T) {
	terminal := []ApplianceStatus{StatusAccepted, StatusRejected, StatusCanceled}
	allStatuses := []ApplianceStatus{StatusPending, StatusProcessed, StatusAccepted, StatusRejected, StatusCanceled}

	for _, from := range terminal {
		for _, to := range allStatuses {
			a := Appliance{Status: from}
			assert.Errorf(t, a.CanTransitionTo(to), "terminal status %q should never allow a transition to %q", from, to)
		}
	}
}
