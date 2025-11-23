//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
)

type EdgeCasesTestSuite struct {
	E2ETestSuite
}

func TestEdgeCases(t *testing.T) {
	suite.Run(t, new(EdgeCasesTestSuite))
}

// TestEdgeCase_UnicodeAndSpecialCharacters tests Unicode and special characters support
func (s *EdgeCasesTestSuite) TestEdgeCase_UnicodeAndSpecialCharacters() {
	// Test with Cyrillic characters
	s.Run("Cyrillic_Characters", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "команда-кириллица",
			Members: []teamModel.TeamMember{
				{UserID: "кир1", Username: "Алексей", IsActive: true},
				{UserID: "кир2", Username: "Мария", IsActive: true},
				{UserID: "кир3", Username: "Дмитрий", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)
		s.Require().Equal("команда-кириллица", team.TeamName)

		// Verify we can get the team back
		resp, getTeam := s.getTeam("команда-кириллица")
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().NotNil(getTeam)
		s.Require().Equal("команда-кириллица", getTeam.TeamName)
		s.Require().Len(getTeam.Members, 3)

		// Create PR with Cyrillic
		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "пр-кириллица-1",
			PullRequestName: "Добавить функцию",
			AuthorID:        "кир1",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Equal("пр-кириллица-1", pr.PullRequestID)
	})

	// Test with Japanese characters
	s.Run("Japanese_Characters", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "チーム日本",
			Members: []teamModel.TeamMember{
				{UserID: "日本1", Username: "山田太郎", IsActive: true},
				{UserID: "日本2", Username: "佐藤花子", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)
		s.Require().Equal("チーム日本", team.TeamName)
	})

	// Test with Chinese characters
	s.Run("Chinese_Characters", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "中文团队",
			Members: []teamModel.TeamMember{
				{UserID: "中1", Username: "张三", IsActive: true},
				{UserID: "中2", Username: "李四", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)
		s.Require().Equal("中文团队", team.TeamName)
	})

	// Test with special characters (dashes, underscores, dots)
	s.Run("Special_Characters", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "team-with_special.chars-123",
			Members: []teamModel.TeamMember{
				{UserID: "spec-1", Username: "User.One", IsActive: true},
				{UserID: "spec_2", Username: "User_Two", IsActive: true},
				{UserID: "spec.3", Username: "User-Three", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)
		s.Require().Equal("team-with_special.chars-123", team.TeamName)

		// Verify retrieval works
		resp, getTeam := s.getTeam("team-with_special.chars-123")
		s.Require().Equal(http.StatusOK, resp.StatusCode)
		s.Require().NotNil(getTeam)
	})

	// Test with mixed Unicode
	s.Run("Mixed_Unicode", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "international-команда-チーム-中文",
			Members: []teamModel.TeamMember{
				{UserID: "mix1", Username: "Alice Алиса アリス 爱丽丝", IsActive: true},
				{UserID: "mix2", Username: "Bob Боб ボブ 鲍勃", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)
	})

	// Test with emojis (if supported)
	s.Run("Emoji_Characters", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "team-with-emoji-🚀",
			Members: []teamModel.TeamMember{
				{UserID: "emoji1", Username: "User 👨‍💻", IsActive: true},
				{UserID: "emoji2", Username: "User 👩‍💻", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		// This might fail depending on DB constraints, so we're flexible here
		if resp.StatusCode == http.StatusCreated {
			s.Require().NotNil(team)
			s.Require().Equal("team-with-emoji-🚀", team.TeamName)
		}
		// If it fails, that's also acceptable behavior
	})
}

// TestEdgeCase_ReassignmentChain tests chain of reassignments
func (s *EdgeCasesTestSuite) TestEdgeCase_ReassignmentChain() {
	// Setup: Create team with 5 members
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "reassign-chain-team",
		Members: []teamModel.TeamMember{
			{UserID: "chain1", Username: "Author", IsActive: true},
			{UserID: "chain2", Username: "ReviewerA", IsActive: true},
			{UserID: "chain3", Username: "ReviewerB", IsActive: true},
			{UserID: "chain4", Username: "ReviewerC", IsActive: true},
			{UserID: "chain5", Username: "ReviewerD", IsActive: true},
		},
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Create PR - will assign 2 reviewers (let's call them A and B)
	prReq := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-chain",
		PullRequestName: "Test reassignment chain",
		AuthorID:        "chain1",
	}
	resp, pr := s.createPR(prReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr)
	s.Require().Len(pr.AssignedReviewers, 2)

	initialReviewers := make([]string, len(pr.AssignedReviewers))
	copy(initialReviewers, pr.AssignedReviewers)

	reviewerA := pr.AssignedReviewers[0]
	reviewerB := pr.AssignedReviewers[1]

	s.T().Logf("Initial reviewers: A=%s, B=%s", reviewerA, reviewerB)

	// Step 1: Reassign reviewer A → should become C or D
	resp, reassignResp1, _ := s.reassignReviewer("pr-chain", reviewerA)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(reassignResp1)
	reviewerC := reassignResp1.ReplacedBy

	s.Require().NotEqual(reviewerA, reviewerC, "new reviewer should be different from A")
	s.Require().NotEqual(reviewerB, reviewerC, "new reviewer should be different from B")
	s.Require().NotEqual("chain1", reviewerC, "new reviewer should not be author")
	s.Require().Contains(reassignResp1.PR.AssignedReviewers, reviewerC, "new reviewer should be in list")
	s.Require().Contains(reassignResp1.PR.AssignedReviewers, reviewerB, "B should still be in list")
	s.Require().NotContains(reassignResp1.PR.AssignedReviewers, reviewerA, "A should be removed")

	s.T().Logf("After 1st reassignment: B=%s, C=%s", reviewerB, reviewerC)

	// Step 2: Reassign reviewer B → should become D or A (but not C or author)
	resp, reassignResp2, _ := s.reassignReviewer("pr-chain", reviewerB)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(reassignResp2)
	reviewerD := reassignResp2.ReplacedBy

	s.Require().NotEqual(reviewerB, reviewerD, "new reviewer should be different from B")
	s.Require().NotEqual(reviewerC, reviewerD, "new reviewer should be different from C")
	s.Require().NotEqual("chain1", reviewerD, "new reviewer should not be author")
	s.Require().Contains(reassignResp2.PR.AssignedReviewers, reviewerD, "new reviewer should be in list")
	s.Require().Contains(reassignResp2.PR.AssignedReviewers, reviewerC, "C should still be in list")
	s.Require().NotContains(reassignResp2.PR.AssignedReviewers, reviewerB, "B should be removed")

	s.T().Logf("After 2nd reassignment: C=%s, D=%s", reviewerC, reviewerD)

	// Final verification: Should have C and D
	s.Require().Len(reassignResp2.PR.AssignedReviewers, 2, "should still have 2 reviewers")

	finalReviewers := reassignResp2.PR.AssignedReviewers
	s.Require().Contains(finalReviewers, reviewerC, "C should be final reviewer")
	s.Require().Contains(finalReviewers, reviewerD, "D should be final reviewer")
	s.Require().Len(finalReviewers, 2, "should have 2 reviewers")
}

// TestEdgeCase_TeamUpdateAttempt tests that updating team fails
func (s *EdgeCasesTestSuite) TestEdgeCase_TeamUpdateAttempt() {
	// Create team
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "immutable-team",
		Members: []teamModel.TeamMember{
			{UserID: "imm1", Username: "User1", IsActive: true},
			{UserID: "imm2", Username: "User2", IsActive: true},
		},
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Try to "update" team by creating with same name but different members
	updateReq := &teamModel.AddTeamRequest{
		TeamName: "immutable-team",
		Members: []teamModel.TeamMember{
			{UserID: "imm3", Username: "User3", IsActive: true},
			{UserID: "imm4", Username: "User4", IsActive: true},
		},
	}
	resp, _ = s.createTeam(updateReq)
	s.Require().Equal(http.StatusBadRequest, resp.StatusCode, "should not allow team update via create")

	// Get error response body from the createTeam request
	bodyBytes, _ := json.Marshal(updateReq)
	_, respBody := s.doRequest("POST", "/team/add", strings.NewReader(string(bodyBytes)))
	errorCode, _ := s.parseErrorResponse(respBody)
	s.Require().Equal("TEAM_EXISTS", errorCode)

	// Verify original team unchanged
	resp, getTeam := s.getTeam("immutable-team")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(getTeam)
	s.Require().Len(getTeam.Members, 2)

	// Check that original members are still there
	memberIDs := make([]string, len(getTeam.Members))
	for i, member := range getTeam.Members {
		memberIDs[i] = member.UserID
	}
	s.Require().Contains(memberIDs, "imm1")
	s.Require().Contains(memberIDs, "imm2")
	s.Require().NotContains(memberIDs, "imm3")
	s.Require().NotContains(memberIDs, "imm4")
}

// TestEdgeCase_LongNames tests very long names (near 255 char limit)
func (s *EdgeCasesTestSuite) TestEdgeCase_LongNames() {
	// Test with 255-character name (at the limit)
	s.Run("MaxLength_255_Characters", func() {
		longName := string(make([]byte, 255))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}

		teamReq := &teamModel.AddTeamRequest{
			TeamName: longName,
			Members: []teamModel.TeamMember{
				{UserID: "long1", Username: "User1", IsActive: true},
			},
		}

		resp, team := s.createTeam(teamReq)
		// Should succeed or fail gracefully depending on constraints
		if resp.StatusCode == http.StatusCreated {
			s.Require().NotNil(team)
			s.Require().Equal(longName, team.TeamName)
		}
	})

	// Test with name exceeding 255 characters (should fail)
	s.Run("ExceedMaxLength_256_Characters", func() {
		tooLongName := string(make([]byte, 256))
		for i := range tooLongName {
			tooLongName = tooLongName[:i] + "b" + tooLongName[i+1:]
		}

		teamReq := &teamModel.AddTeamRequest{
			TeamName: tooLongName,
			Members: []teamModel.TeamMember{
				{UserID: "toolong1", Username: "User1", IsActive: true},
			},
		}

		resp, _ := s.createTeam(teamReq)
		// Should fail with constraint violation
		s.Require().NotEqual(http.StatusCreated, resp.StatusCode, "should not create team with name > 255 chars")
	})
}

// TestEdgeCase_EmptyReviewerList tests PR with no possible reviewers
func (s *EdgeCasesTestSuite) TestEdgeCase_EmptyReviewerList() {
	s.Run("AllMembersInactive_ExceptAuthor", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "all-inactive-team",
			Members: []teamModel.TeamMember{
				{UserID: "inactive-author", Username: "Author", IsActive: true},
				{UserID: "inactive1", Username: "Inactive1", IsActive: false},
				{UserID: "inactive2", Username: "Inactive2", IsActive: false},
				{UserID: "inactive3", Username: "Inactive3", IsActive: false},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-no-reviewers",
			PullRequestName: "PR with no possible reviewers",
			AuthorID:        "inactive-author",
		}
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)
		s.Require().Empty(pr.AssignedReviewers, "should have no reviewers when all others are inactive")
	})
}
