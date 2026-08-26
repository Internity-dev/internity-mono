package review

import (
	"context"
	"testing"

	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type fakePlacementChecker struct {
	hasPlacement bool
	err          error
}

func (f fakePlacementChecker) HasPlacementAtCompany(ctx context.Context, userID string, companyID int64) (bool, error) {
	return f.hasPlacement, f.err
}

// Regression coverage for a real, previously-exploitable bug: a mentor could
// read (and write) another company's confidential student reviews just by
// knowing a student's UUID, because ListReviewsForUser/CreateReview had no
// company-ownership check for staff roles — only students were scoped.
func TestListReviewsForUser_MentorScopedToOwnCompany(t *testing.T) {
	companyID := int64(1)
	mentor := &identity.User{ID: "mentor-1", Role: identity.RoleMentor, CompanyID: &companyID}

	t.Run("mentor with no placement relationship to the student is forbidden", func(t *testing.T) {
		svc := &Service{placements: fakePlacementChecker{hasPlacement: false}}
		_, err := svc.ListReviewsForUser(context.Background(), mentor, "student-at-a-different-company")
		require.Error(t, err)
		assert.Equal(t, errForbidden, err)
	})

	t.Run("mentor with no company assigned at all is forbidden", func(t *testing.T) {
		bareMentor := &identity.User{ID: "mentor-2", Role: identity.RoleMentor}
		svc := &Service{placements: fakePlacementChecker{hasPlacement: true}}
		_, err := svc.ListReviewsForUser(context.Background(), bareMentor, "student-1")
		require.Error(t, err)
		assert.Equal(t, errForbidden, err)
	})
}

func TestAssertMentorMentorsStudent_AllowsRealPlacement(t *testing.T) {
	companyID := int64(1)
	mentor := &identity.User{ID: "mentor-1", Role: identity.RoleMentor, CompanyID: &companyID}
	svc := &Service{placements: fakePlacementChecker{hasPlacement: true}}

	err := svc.assertMentorMentorsStudent(context.Background(), mentor, "student-actually-at-company-1")
	assert.NoError(t, err)
}

func TestCreateReview_MentorScopedToOwnCompany(t *testing.T) {
	companyID := int64(1)
	mentor := &identity.User{ID: "mentor-1", Role: identity.RoleMentor, CompanyID: &companyID}
	studentID := "student-at-a-different-company"

	svc := &Service{placements: fakePlacementChecker{hasPlacement: false}}
	_, err := svc.CreateReview(context.Background(), mentor, CreateReviewInput{
		RevieweeType:   RevieweeUser,
		RevieweeUserID: &studentID,
		Rating:         4,
	})
	require.Error(t, err)
	assert.Equal(t, errForbidden, err)
}
