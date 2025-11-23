//go:build e2e
// +build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
)

type ErrorScenariosTestSuite struct {
	E2ETestSuite
}

func TestErrorScenarios(t *testing.T) {
	suite.Run(t, new(ErrorScenariosTestSuite))
}

// TestScenario4_NoCandidateError tests NO_CANDIDATE error scenarios
// Scenario 4: Test reassignment when no candidates available
func (s *ErrorScenariosTestSuite) TestScenario4_NoCandidateError() {
	// Sub-scenario 4.1: Basic case - 2 members, try to reassign the only reviewer
	s.Run("4.1_TwoMembersTeam_NoCandidateForReassignment", func() {
		// Step 1: Create team with 2 members
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "minimal-team",
			Members: []teamModel.TeamMember{
				{UserID: "min1", Username: "Author", IsActive: true},
				{UserID: "min2", Username: "OnlyReviewer", IsActive: true},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		// Step 2: Create PR → 1 reviewer assigned
		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-nocandidate-1",
			PullRequestName: "Test no candidate",
			AuthorID:        "min1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Len(pr.AssignedReviewers, 1)
		s.Require().Equal("min2", pr.AssignedReviewers[0])

		// Step 3: Try to reassign the only reviewer → NO_CANDIDATE error
		resp, _, respBody := s.reassignReviewer("pr-nocandidate-1", "min2")
		s.Require().Equal(http.StatusConflict, resp.StatusCode)

		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NO_CANDIDATE", errorCode)
	})

	// Sub-scenario 4.2: Extended case - deactivate members after PR creation
	s.Run("4.2_DeactivateMembersAfterCreation_NoCandidate", func() {
		// Step 1: Create team with 4 members
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "shrinking-team",
			Members: []teamModel.TeamMember{
				{UserID: "shr1", Username: "Author", IsActive: true},
				{UserID: "shr2", Username: "Reviewer1", IsActive: true},
				{UserID: "shr3", Username: "Reviewer2", IsActive: true},
				{UserID: "shr4", Username: "Member4", IsActive: true},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		// Step 2: Create PR → 2 reviewers assigned
		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-nocandidate-2",
			PullRequestName: "Test shrinking team",
			AuthorID:        "shr1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Len(pr.AssignedReviewers, 2)

		// Step 3: Deactivate all except author and one reviewer
		oneReviewer := pr.AssignedReviewers[0]

		// Deactivate all members except author and oneReviewer
		for _, memberID := range []string{"shr2", "shr3", "shr4"} {
			if memberID != oneReviewer {
				resp, user := s.setUserActive(memberID, false)
				s.Require().Equal(http.StatusOK, resp.StatusCode)
				s.Require().NotNil(user)
			}
		}

		// Step 4: Try to reassign → NO_CANDIDATE
		resp, _, respBody := s.reassignReviewer("pr-nocandidate-2", oneReviewer)
		s.Require().Equal(http.StatusConflict, resp.StatusCode)

		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NO_CANDIDATE", errorCode)
	})
}

// TestScenario5_NotAssignedError tests NOT_ASSIGNED error scenario
// Scenario 5: Try to reassign a user who is not assigned as reviewer
func (s *ErrorScenariosTestSuite) TestScenario5_NotAssignedError() {
	// Step 1: Create team with 5 members
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "test-team-notassigned",
		Members: []teamModel.TeamMember{
			{UserID: "tna1", Username: "Author", IsActive: true},
			{UserID: "tna2", Username: "Member2", IsActive: true},
			{UserID: "tna3", Username: "Member3", IsActive: true},
			{UserID: "tna4", Username: "Member4", IsActive: true},
			{UserID: "tna5", Username: "Member5", IsActive: true},
		},
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Step 2: Create PR (2 reviewers will be automatically assigned)
	prReq := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-notassigned",
		PullRequestName: "Test not assigned",
		AuthorID:        "tna1",
	}
	resp, pr := s.createPR(prReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr)
	s.Require().Len(pr.AssignedReviewers, 2)

	// Step 3: Find a team member who was NOT assigned
	var notAssignedMember string
	for _, memberID := range []string{"tna2", "tna3", "tna4", "tna5"} {
		if !contains(pr.AssignedReviewers, memberID) {
			notAssignedMember = memberID
			break
		}
	}
	s.Require().NotEmpty(notAssignedMember, "should have at least one non-assigned member")

	// Step 4: Try to reassign the non-assigned member → NOT_ASSIGNED error
	resp, _, respBody := s.reassignReviewer("pr-notassigned", notAssignedMember)
	s.Require().Equal(http.StatusConflict, resp.StatusCode)

	errorCode, _ := s.parseErrorResponse(respBody)
	s.Require().Equal("NOT_ASSIGNED", errorCode)
}

// TestScenario6_MultiplePRsAndGetReview tests multiple PRs and getReview endpoint
// Scenario 6: Create multiple PRs, merge some, and verify getReview returns all PRs for a reviewer
func (s *ErrorScenariosTestSuite) TestScenario6_MultiplePRsAndGetReview() {
	// Step 1: Create team with 5 members
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "multi-pr-team",
		Members: []teamModel.TeamMember{
			{UserID: "mpr1", Username: "User1", IsActive: true},
			{UserID: "mpr2", Username: "User2", IsActive: true},
			{UserID: "mpr3", Username: "User3", IsActive: true},
			{UserID: "mpr4", Username: "User4", IsActive: true},
			{UserID: "mpr5", Username: "User5", IsActive: true},
		},
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Step 2: Create PR1 from mpr1
	pr1Req := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-multi-1",
		PullRequestName: "First PR",
		AuthorID:        "mpr1",
	}
	resp, pr1 := s.createPR(pr1Req)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr1)

	// Step 3: Create PR2 from mpr2
	pr2Req := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-multi-2",
		PullRequestName: "Second PR",
		AuthorID:        "mpr2",
	}
	resp, pr2 := s.createPR(pr2Req)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr2)

	// Step 4: Create PR3 from mpr5
	pr3Req := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-multi-3",
		PullRequestName: "Third PR",
		AuthorID:        "mpr5",
	}
	resp, pr3 := s.createPR(pr3Req)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr3)

	// Step 5: Merge PR1
	resp, mergedPR1 := s.mergePR("pr-multi-1")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(mergedPR1)
	s.Require().Equal("MERGED", mergedPR1.Status)

	// Step 6: Get review list for mpr2 (who should be reviewer in some PRs)
	resp, reviewResp := s.getUserReviews("mpr2")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(reviewResp)
	s.Require().Equal("mpr2", reviewResp.UserID)

	// Step 7: CRITICAL - Verify that getReview returns ALL PRs where mpr2 is reviewer
	// This includes both OPEN and MERGED PRs
	// Check which PRs mpr2 was assigned to based on API responses
	var expectedPRs []string
	if contains(pr1.AssignedReviewers, "mpr2") {
		expectedPRs = append(expectedPRs, "pr-multi-1")
	}
	if contains(pr2.AssignedReviewers, "mpr2") {
		expectedPRs = append(expectedPRs, "pr-multi-2")
	}
	if contains(pr3.AssignedReviewers, "mpr2") {
		expectedPRs = append(expectedPRs, "pr-multi-3")
	}

	// Verify the response contains all expected PRs
	s.Require().Len(reviewResp.PullRequests, len(expectedPRs), "should return all PRs where mpr2 is reviewer")

	var foundPRs []string
	for _, pr := range reviewResp.PullRequests {
		foundPRs = append(foundPRs, pr.PullRequestID)
		// Verify status is valid (OPEN or MERGED)
		s.Require().Contains([]string{"OPEN", "MERGED"}, pr.Status, "status should be OPEN or MERGED for PR %s", pr.PullRequestID)
	}

	for _, prID := range expectedPRs {
		s.Require().Contains(foundPRs, prID, "should include PR %s in review list", prID)
	}

	// Step 8: Verify statuses match expected values
	// pr-multi-1 was merged, others are OPEN
	for _, pr := range reviewResp.PullRequests {
		if pr.PullRequestID == "pr-multi-1" {
			s.Require().Equal("MERGED", pr.Status, "pr-multi-1 should be MERGED")
		} else {
			s.Require().Equal("OPEN", pr.Status, "PR %s should be OPEN", pr.PullRequestID)
		}
	}

	// Step 9: Get review for user with no PRs (mpr1 is author of pr1, should not be reviewer)
	resp, emptyReviewResp := s.getUserReviews("mpr1")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(emptyReviewResp)
	s.Require().Equal("mpr1", emptyReviewResp.UserID)

	// mpr1 is author of pr1, should not be reviewer (verify from API responses)
	s.Require().NotContains(pr1.AssignedReviewers, "mpr1", "mpr1 should not be reviewer in pr1 (is author)")
	// If mpr1 is not reviewer in any PR, list should be empty
	// Otherwise, verify it matches actual assignments from API responses
	var expectedMpr1PRs []string
	if contains(pr2.AssignedReviewers, "mpr1") {
		expectedMpr1PRs = append(expectedMpr1PRs, "pr-multi-2")
	}
	if contains(pr3.AssignedReviewers, "mpr1") {
		expectedMpr1PRs = append(expectedMpr1PRs, "pr-multi-3")
	}
	s.Require().Len(emptyReviewResp.PullRequests, len(expectedMpr1PRs), "should return correct number of PRs for mpr1")
}

// Helper function to check if slice contains value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
