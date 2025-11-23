//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
)

type BusinessScenariosTestSuite struct {
	E2ETestSuite
}

func TestBusinessScenarios(t *testing.T) {
	suite.Run(t, new(BusinessScenariosTestSuite))
}

// TestScenario1_FullPRLifecycle tests the complete PR lifecycle
// Scenario 1: Create team → Create PR → Auto-assign reviewers → Reassign → Merge → Check idempotency → Try reassign after merge
func (s *BusinessScenariosTestSuite) TestScenario1_FullPRLifecycle() {
	// Step 1: Create team with 5 active members
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "engineering",
		Members: []teamModel.TeamMember{
			{UserID: "eng1", Username: "Alice", IsActive: true},
			{UserID: "eng2", Username: "Bob", IsActive: true},
			{UserID: "eng3", Username: "Charlie", IsActive: true},
			{UserID: "eng4", Username: "David", IsActive: true},
			{UserID: "eng5", Username: "Eve", IsActive: true},
		},
	}
	resp, team := s.createTeam(teamReq)
	if resp.StatusCode != http.StatusCreated {
		// Get full error response for debugging
		bodyBytes, _ := json.Marshal(teamReq)
		_, respBody := s.doRequest("POST", "/team/add", strings.NewReader(string(bodyBytes)))
		s.T().Logf("Failed to create team. Status: %d, Request: %s, Response: %s",
			resp.StatusCode, string(bodyBytes), string(respBody))
	}
	s.Require().Equal(http.StatusCreated, resp.StatusCode, "team creation should succeed")
	s.Require().NotNil(team)
	s.Require().Equal("engineering", team.TeamName)
	s.Require().Len(team.Members, 5)

	// Verify team was created in database
	var teamCount int64
	s.db.Table("teams").Where("team_name = ?", "engineering").Count(&teamCount)
	s.T().Logf("Team 'engineering' exists in DB: %d records", teamCount)

	var userCount int64
	s.db.Table("users").Where("team_name = ?", "engineering").Count(&userCount)
	s.T().Logf("Users in 'engineering' team: %d", userCount)

	// Step 2: Create PR from eng1
	prReq := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-lifecycle-1",
		PullRequestName: "Add new feature",
		AuthorID:        "eng1",
	}
	resp, pr := s.createPR(prReq)
	if resp.StatusCode != http.StatusCreated {
		// Check if author exists in database
		var authorExists bool
		s.db.Table("users").Where("user_id = ?", "eng1").Select("1").Limit(1).Scan(&authorExists)
		s.T().Logf("Author 'eng1' exists in DB: %v", authorExists)

		// Get team for author
		var authorTeam string
		s.db.Table("users").Where("user_id = ?", "eng1").Pluck("team_name", &authorTeam)
		s.T().Logf("Author 'eng1' team: %s", authorTeam)

		// Get active users in team
		var activeUsers []string
		s.db.Table("users").Where("team_name = ? AND is_active = ?", "engineering", true).Pluck("user_id", &activeUsers)
		s.T().Logf("Active users in 'engineering' team: %v", activeUsers)

		// Get application logs for debugging
		appLogs := s.getAppLogs()
		if len(appLogs) > 0 {
			// Show last 2000 characters (more detailed)
			start := len(appLogs) - 2000
			if start < 0 {
				start = 0
			}
			s.T().Logf("Application logs (last 2000 chars):\n%s", appLogs[start:])
		}
	}
	s.Require().Equal(http.StatusCreated, resp.StatusCode, "PR creation should succeed")
	s.Require().NotNil(pr)
	s.Require().Equal("pr-lifecycle-1", pr.PullRequestID)
	s.Require().Equal("OPEN", pr.Status)

	// Step 3: Check automatic reviewer assignment (2 reviewers, not author)
	s.Require().Len(pr.AssignedReviewers, 2, "should assign exactly 2 reviewers")
	s.Require().NotContains(pr.AssignedReviewers, "eng1", "author should not be assigned as reviewer")
	s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[0], "reviewer should not be the author")
	s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[1], "reviewer should not be the author")

	// Step 4: Reassign one reviewer
	oldReviewer := pr.AssignedReviewers[0]
	resp, reassignResp, _ := s.reassignReviewer("pr-lifecycle-1", oldReviewer)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(reassignResp)

	// Step 5: Verify new reviewer is from same team and different from old one
	s.Require().NotEqual(oldReviewer, reassignResp.ReplacedBy, "new reviewer should be different")
	s.Require().NotContains(reassignResp.PR.AssignedReviewers, oldReviewer, "old reviewer should be removed")
	s.Require().Contains(reassignResp.PR.AssignedReviewers, reassignResp.ReplacedBy, "new reviewer should be in the list")
	s.Require().NotEqual("eng1", reassignResp.ReplacedBy, "new reviewer should not be the author")
	s.Require().NotEqual(reassignResp.PR.AuthorID, reassignResp.ReplacedBy, "new reviewer should not be the author")

	// Step 6: Merge PR
	resp, mergedPR := s.mergePR("pr-lifecycle-1")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(mergedPR)
	s.Require().Equal("MERGED", mergedPR.Status)
	s.Require().NotNil(mergedPR.MergedAt)

	firstMergedAt := mergedPR.MergedAt
	firstReviewers := mergedPR.AssignedReviewers

	// Step 7: Check merge idempotency (repeat merge)
	time.Sleep(100 * time.Millisecond)
	resp, mergedPR2 := s.mergePR("pr-lifecycle-1")
	s.Require().Equal(http.StatusOK, resp.StatusCode, "merge should be idempotent")
	s.Require().NotNil(mergedPR2)
	s.Require().Equal("MERGED", mergedPR2.Status)
	s.Require().Equal(firstMergedAt, mergedPR2.MergedAt, "mergedAt should not change on repeated merge")
	s.Require().Equal(firstReviewers, mergedPR2.AssignedReviewers, "reviewers should not change on repeated merge")

	// Step 8: Try to reassign after merge - should fail with PR_MERGED
	resp, _, respBody := s.reassignReviewer("pr-lifecycle-1", mergedPR2.AssignedReviewers[0])
	s.Require().Equal(http.StatusConflict, resp.StatusCode, "should not allow reassignment after merge")

	// Parse error response
	errorCode, errorMsg := s.parseErrorResponse(respBody)
	s.T().Logf("Reassign after merge error - Code: %s, Message: %s", errorCode, errorMsg)
	s.Require().Equal("PR_MERGED", errorCode, "should return PR_MERGED error code")
}

// TestScenario2_ActivityManagement tests activity management and its effect on assignment
// Scenario 2: Create team with mixed activity → Create PR → Check only active assigned →
// Deactivate reviewer → Create new PR → Check deactivated not assigned → Reactivate → Create PR → Check can be assigned
func (s *BusinessScenariosTestSuite) TestScenario2_ActivityManagement() {
	// Step 1: Create team with mixed active/inactive members
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "mixed-team",
		Members: []teamModel.TeamMember{
			{UserID: "mix1", Username: "Active1", IsActive: true},
			{UserID: "mix2", Username: "Active2", IsActive: true},
			{UserID: "mix3", Username: "Active3", IsActive: true},
			{UserID: "mix4", Username: "Inactive1", IsActive: false},
			{UserID: "mix5", Username: "Inactive2", IsActive: false},
		},
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Step 2: Create PR from active member
	prReq := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-activity-1",
		PullRequestName: "Test active assignment",
		AuthorID:        "mix1",
	}
	resp, pr := s.createPR(prReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr)

	// Step 3: CRITICAL - Check that ONLY active members are assigned (not > 2 because of inactive)
	// Should be exactly 2 reviewers (mix2 and mix3), NOT including inactive members
	s.Require().Len(pr.AssignedReviewers, 2, "should assign exactly 2 active reviewers")
	s.Require().NotContains(pr.AssignedReviewers, "mix1", "author should not be reviewer")
	s.Require().NotContains(pr.AssignedReviewers, "mix4", "inactive member should not be assigned")
	s.Require().NotContains(pr.AssignedReviewers, "mix5", "inactive member should not be assigned")
	s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[0], "reviewer should not be the author")
	s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[1], "reviewer should not be the author")

	// Step 4: Deactivate one of the reviewers
	deactivatedReviewer := pr.AssignedReviewers[0]
	resp, user := s.setUserActive(deactivatedReviewer, false)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(user)
	s.Require().False(user.IsActive)

	// Step 5: Create new PR
	prReq2 := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-activity-2",
		PullRequestName: "Test after deactivation",
		AuthorID:        "mix1",
	}
	resp, pr2 := s.createPR(prReq2)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr2)

	// Step 6: Verify deactivated member is NOT assigned
	s.Require().NotContains(pr2.AssignedReviewers, deactivatedReviewer, "deactivated member should not be assigned")
	s.Require().NotEqual(pr2.AuthorID, pr2.AssignedReviewers[0], "reviewer should not be the author")

	// Should only have 1 reviewer now (only mix2 or mix3, whoever wasn't deactivated, plus author mix1 excluded)
	// Actually, we have mix1 (author), deactivated reviewer, and one active = 3 active members total
	// So should assign 1 reviewer (the one active non-author)
	s.Require().Len(pr2.AssignedReviewers, 1, "should assign 1 reviewer (only one active non-author)")

	// Step 7: Reactivate the user
	resp, user = s.setUserActive(deactivatedReviewer, true)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(user)
	s.Require().True(user.IsActive)

	// Step 8: Create new PR
	prReq3 := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-activity-3",
		PullRequestName: "Test after reactivation",
		AuthorID:        "mix1",
	}
	resp, pr3 := s.createPR(prReq3)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr3)

	// Step 9: Verify reactivated member CAN be assigned now
	// Now we have mix1 (author), mix2, mix3 active - should assign 2 reviewers
	s.Require().Len(pr3.AssignedReviewers, 2, "should assign 2 reviewers after reactivation")
	s.Require().NotContains(pr3.AssignedReviewers, "mix1", "author should not be reviewer")
	s.Require().NotEqual(pr3.AuthorID, pr3.AssignedReviewers[0], "reviewer should not be the author")
	s.Require().NotEqual(pr3.AuthorID, pr3.AssignedReviewers[1], "reviewer should not be the author")
}

// TestScenario3_ReviewerCountLimits tests reviewer count limits with different team sizes
// Scenario 3: Test 4 sub-scenarios with different team compositions
func (s *BusinessScenariosTestSuite) TestScenario3_ReviewerCountLimits() {
	// Sub-scenario 3.1: Team with only 1 member (author only) → 0 reviewers
	s.Run("3.1_TeamWithOnlyAuthor_ZeroReviewers", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "solo-team",
			Members: []teamModel.TeamMember{
				{UserID: "solo1", Username: "SoloMember", IsActive: true},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-solo",
			PullRequestName: "Solo PR",
			AuthorID:        "solo1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Empty(pr.AssignedReviewers, "should have 0 reviewers when team has only author")
	})

	// Sub-scenario 3.2: Team with 2 members → 1 reviewer
	s.Run("3.2_TeamWithTwoMembers_OneReviewer", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "duo-team",
			Members: []teamModel.TeamMember{
				{UserID: "duo1", Username: "Member1", IsActive: true},
				{UserID: "duo2", Username: "Member2", IsActive: true},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-duo",
			PullRequestName: "Duo PR",
			AuthorID:        "duo1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Len(pr.AssignedReviewers, 1, "should have 1 reviewer when team has 2 members")
		s.Require().Equal("duo2", pr.AssignedReviewers[0], "should assign the only other member")
		s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[0], "reviewer should not be the author")
	})

	// Sub-scenario 3.3: Team with 3+ members → 2 reviewers
	s.Run("3.3_TeamWithThreePlusMembers_TwoReviewers", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "large-team",
			Members: []teamModel.TeamMember{
				{UserID: "large1", Username: "Member1", IsActive: true},
				{UserID: "large2", Username: "Member2", IsActive: true},
				{UserID: "large3", Username: "Member3", IsActive: true},
				{UserID: "large4", Username: "Member4", IsActive: true},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-large",
			PullRequestName: "Large team PR",
			AuthorID:        "large1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Len(pr.AssignedReviewers, 2, "should have 2 reviewers when team has 3+ members")
		s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[0], "reviewer should not be the author")
		s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[1], "reviewer should not be the author")
	})

	// Sub-scenario 3.4: Team with 4 members but 2 inactive → 1 reviewer
	s.Run("3.4_TeamWithInactiveMembers_OneReviewer", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "partial-team",
			Members: []teamModel.TeamMember{
				{UserID: "part1", Username: "Author", IsActive: true},
				{UserID: "part2", Username: "ActiveReviewer", IsActive: true},
				{UserID: "part3", Username: "Inactive1", IsActive: false},
				{UserID: "part4", Username: "Inactive2", IsActive: false},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-partial",
			PullRequestName: "Partial team PR",
			AuthorID:        "part1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Len(pr.AssignedReviewers, 1, "should have 1 reviewer (only one active non-author)")
		s.Require().Equal("part2", pr.AssignedReviewers[0], "should assign the only active non-author member")
		s.Require().NotEqual(pr.AuthorID, pr.AssignedReviewers[0], "reviewer should not be the author")
	})
}
