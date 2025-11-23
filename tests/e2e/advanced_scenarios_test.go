//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

type AdvancedScenariosTestSuite struct {
	E2ETestSuite
}

func TestAdvancedScenarios(t *testing.T) {
	suite.Run(t, new(AdvancedScenariosTestSuite))
}

// TestScenario7_ConcurrentPRCreation tests concurrent PR creation for race conditions
// Scenario 7: Create multiple PRs concurrently and verify correctness
func (s *AdvancedScenariosTestSuite) TestScenario7_ConcurrentPRCreation() {
	// Step 1: Create team with 10 members
	members := make([]teamModel.TeamMember, 10)
	for i := 0; i < 10; i++ {
		members[i] = teamModel.TeamMember{
			UserID:   fmt.Sprintf("conc%d", i+1),
			Username: fmt.Sprintf("User%d", i+1),
			IsActive: true,
		}
	}

	teamReq := &teamModel.AddTeamRequest{
		TeamName: "concurrent-team",
		Members:  members,
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Step 2: Create 5 PRs concurrently from different authors
	numPRs := 5
	var wg sync.WaitGroup
	results := make(chan struct {
		prID   string
		status int
		pr     *pullrequestModel.PullRequestResponse
		err    error
	}, numPRs)

	for i := 0; i < numPRs; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			prReq := &pullrequestModel.CreatePullRequestRequest{
				PullRequestID:   fmt.Sprintf("pr-concurrent-%d", index),
				PullRequestName: fmt.Sprintf("Concurrent PR %d", index),
				AuthorID:        fmt.Sprintf("conc%d", index+1),
			}

			resp, pr, err := s.createPRNoFail(prReq)
			results <- struct {
				prID   string
				status int
				pr     *pullrequestModel.PullRequestResponse
				err    error
			}{
				prID:   prReq.PullRequestID,
				status: resp.StatusCode,
				pr:     pr,
				err:    err,
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Step 3: Verify all PRs were created successfully
	createdPRs := make(map[string]*pullrequestModel.PullRequestResponse)
	for result := range results {
		s.Require().NoError(result.err, "PR %s should be created without error", result.prID)
		s.Require().Equal(http.StatusCreated, result.status, "PR %s should be created successfully", result.prID)
		s.Require().NotNil(result.pr, "PR %s should have response", result.prID)
		createdPRs[result.prID] = result.pr
	}
	s.Require().Len(createdPRs, numPRs, "all PRs should be created")

	// Step 4: Verify each PR has correct number of reviewers
	for prID, pr := range createdPRs {
		s.Require().LessOrEqual(len(pr.AssignedReviewers), 2, "PR %s should have at most 2 reviewers", prID)
		s.Require().GreaterOrEqual(len(pr.AssignedReviewers), 0, "PR %s should have 0-2 reviewers", prID)

		// Verify no duplicate reviewers in the same PR
		reviewerMap := make(map[string]bool)
		for _, reviewer := range pr.AssignedReviewers {
			s.Require().False(reviewerMap[reviewer], "PR %s should not have duplicate reviewer %s", prID, reviewer)
			reviewerMap[reviewer] = true
		}

		// Verify reviewers are not the author
		for _, reviewer := range pr.AssignedReviewers {
			s.Require().NotEqual(pr.AuthorID, reviewer, "PR %s: reviewer should not be author", prID)
		}
	}

	// Step 5: Check fair distribution (no one reviewer assigned to all PRs)
	reviewerCounts := make(map[string]int)
	for _, pr := range createdPRs {
		for _, reviewer := range pr.AssignedReviewers {
			reviewerCounts[reviewer]++
		}
	}

	// With 5 PRs and 10 members, no single reviewer should have all assignments
	// This is a soft check - just ensure it's somewhat distributed
	for reviewer, count := range reviewerCounts {
		s.Require().Less(count, numPRs, "reviewer %s should not be assigned to all PRs", reviewer)
	}
}

// TestScenario8_MergeIdempotency tests deep merge idempotency
// Scenario 8: Merge PR, wait, merge again, verify nothing changed
func (s *AdvancedScenariosTestSuite) TestScenario8_MergeIdempotency() {
	// Step 1: Create team
	teamReq := &teamModel.AddTeamRequest{
		TeamName: "idempotent-team",
		Members: []teamModel.TeamMember{
			{UserID: "idemp1", Username: "User1", IsActive: true},
			{UserID: "idemp2", Username: "User2", IsActive: true},
			{UserID: "idemp3", Username: "User3", IsActive: true},
		},
	}
	resp, team := s.createTeam(teamReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(team)

	// Step 2: Create PR
	prReq := &pullrequestModel.CreatePullRequestRequest{
		PullRequestID:   "pr-idempotent",
		PullRequestName: "Test idempotency",
		AuthorID:        "idemp1",
	}
	resp, pr := s.createPR(prReq)
	s.Require().Equal(http.StatusCreated, resp.StatusCode)
	s.Require().NotNil(pr)

	// Step 3: First merge - capture state
	resp, mergedPR1 := s.mergePR("pr-idempotent")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NotNil(mergedPR1)
	s.Require().Equal("MERGED", mergedPR1.Status)
	s.Require().NotNil(mergedPR1.MergedAt)

	firstMergedAt := mergedPR1.MergedAt
	firstReviewers := make([]string, len(mergedPR1.AssignedReviewers))
	copy(firstReviewers, mergedPR1.AssignedReviewers)
	firstCreatedAt := mergedPR1.CreatedAt

	// Step 4: Wait 100ms
	time.Sleep(100 * time.Millisecond)

	// Step 5: Second merge - verify idempotency
	resp, mergedPR2 := s.mergePR("pr-idempotent")
	s.Require().Equal(http.StatusOK, resp.StatusCode, "repeated merge should return 200")
	s.Require().NotNil(mergedPR2)
	s.Require().Equal("MERGED", mergedPR2.Status)

	// CRITICAL checks for true idempotency
	s.Require().Equal(firstMergedAt, mergedPR2.MergedAt, "mergedAt timestamp should NOT change")
	s.Require().Equal(firstReviewers, mergedPR2.AssignedReviewers, "reviewers should NOT change")
	s.Require().Equal(firstCreatedAt, mergedPR2.CreatedAt, "createdAt should NOT change")

	// Step 6: Third merge to be absolutely sure
	time.Sleep(50 * time.Millisecond)
	resp, mergedPR3 := s.mergePR("pr-idempotent")
	s.Require().Equal(http.StatusOK, resp.StatusCode)
	s.Require().Equal(firstMergedAt, mergedPR3.MergedAt, "mergedAt should remain the same on 3rd merge")
	s.Require().Equal(firstReviewers, mergedPR3.AssignedReviewers, "reviewers should remain the same on 3rd merge")
	s.Require().Equal(firstCreatedAt, mergedPR3.CreatedAt, "createdAt should remain the same on 3rd merge")
}

// TestScenario9_DuplicateKeysError tests duplicate team and PR creation
// Scenario 9: Try to create duplicate teams and PRs
func (s *AdvancedScenariosTestSuite) TestScenario9_DuplicateKeysError() {
	// Test 9.1: Duplicate team
	s.Run("9.1_DuplicateTeam_TEAM_EXISTS", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "unique-team",
			Members: []teamModel.TeamMember{
				{UserID: "uniq1", Username: "User1", IsActive: true},
			},
		}

		// First creation - should succeed
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		// Second creation with same name - should fail
		resp, _ = s.createTeam(teamReq)
		s.Require().Equal(http.StatusBadRequest, resp.StatusCode)

		// Get error response body from the createTeam request
		bodyBytes, _ := json.Marshal(teamReq)
		_, respBody := s.doRequest("POST", "/team/add", strings.NewReader(string(bodyBytes)))
		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("TEAM_EXISTS", errorCode)
	})

	// Test 9.2: Duplicate PR
	s.Run("9.2_DuplicatePR_PR_EXISTS", func() {
		teamReq := &teamModel.AddTeamRequest{
			TeamName: "pr-dup-team",
			Members: []teamModel.TeamMember{
				{UserID: "prdup1", Username: "User1", IsActive: true},
				{UserID: "prdup2", Username: "User2", IsActive: true},
			},
		}
		resp, team := s.createTeam(teamReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(team)

		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-duplicate",
			PullRequestName: "Duplicate PR test",
			AuthorID:        "prdup1",
		}

		// First creation - should succeed
		resp, pr := s.createPR(prReq)
		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().NotNil(pr)

		// Second creation with same ID - should fail
		resp, _ = s.createPR(prReq)
		s.Require().Equal(http.StatusConflict, resp.StatusCode)

		// Get error response body from the createPR request
		bodyBytes, _ := json.Marshal(prReq)
		_, respBody := s.doRequest("POST", "/pullRequest/create", strings.NewReader(string(bodyBytes)))
		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("PR_EXISTS", errorCode)
	})
}

// TestScenario10_NotFoundErrors tests NOT_FOUND error handling
// Scenario 10: Test various NOT_FOUND scenarios
func (s *AdvancedScenariosTestSuite) TestScenario10_NotFoundErrors() {
	// Sub-test 10.1: Get non-existent team
	s.Run("10.1_GetNonExistentTeam_NOT_FOUND", func() {
		resp, _ := s.getTeam("nonexistent-team")
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)

		_, respBody := s.doRequest("GET", "/team/get?team_name=nonexistent-team", nil)
		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NOT_FOUND", errorCode)
	})

	// Sub-test 10.2: Create PR with non-existent author
	s.Run("10.2_CreatePRWithNonExistentAuthor_NOT_FOUND", func() {
		prReq := &pullrequestModel.CreatePullRequestRequest{
			PullRequestID:   "pr-noauthor",
			PullRequestName: "PR with no author",
			AuthorID:        "nonexistent-author",
		}

		resp, _ := s.createPR(prReq)
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)

		// Get error response body from the createPR request
		bodyBytes, _ := json.Marshal(prReq)
		_, respBody := s.doRequest("POST", "/pullRequest/create", strings.NewReader(string(bodyBytes)))
		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NOT_FOUND", errorCode)
	})

	// Sub-test 10.3: Merge non-existent PR
	s.Run("10.3_MergeNonExistentPR_NOT_FOUND", func() {
		resp, _ := s.mergePR("nonexistent-pr")
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)

		// Get error response body from the mergePR request
		mergeReq := pullrequestModel.MergePullRequestRequest{
			PullRequestID: "nonexistent-pr",
		}
		bodyBytes, _ := json.Marshal(mergeReq)
		_, respBody := s.doRequest("POST", "/pullRequest/merge", strings.NewReader(string(bodyBytes)))
		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NOT_FOUND", errorCode)
	})

	// Sub-test 10.4: Reassign in non-existent PR
	s.Run("10.4_ReassignInNonExistentPR_NOT_FOUND", func() {
		resp, _, respBody := s.reassignReviewer("nonexistent-pr", "some-user")
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)

		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NOT_FOUND", errorCode)
	})

	// Sub-test 10.5: setIsActive for non-existent user
	s.Run("10.5_SetIsActiveNonExistentUser_NOT_FOUND", func() {
		resp, _ := s.setUserActive("nonexistent-user", false)
		s.Require().Equal(http.StatusNotFound, resp.StatusCode)

		// Get error response body from the setUserActive request
		setActiveReq := userModel.SetIsActiveRequest{
			UserID:   "nonexistent-user",
			IsActive: false,
		}
		bodyBytes, _ := json.Marshal(setActiveReq)
		_, respBody := s.doRequest("POST", "/users/setIsActive", strings.NewReader(string(bodyBytes)))
		errorCode, _ := s.parseErrorResponse(respBody)
		s.Require().Equal("NOT_FOUND", errorCode)
	})

	// Sub-test 10.6: getReview for non-existent user - special case (returns empty list)
	s.Run("10.6_GetReviewNonExistentUser_EmptyList", func() {
		resp, reviewResp := s.getUserReviews("nonexistent-user")
		s.Require().Equal(http.StatusOK, resp.StatusCode, "getReview should return 200 even for non-existent user")
		s.Require().NotNil(reviewResp)
		s.Require().Equal("nonexistent-user", reviewResp.UserID)
		s.Require().Empty(reviewResp.PullRequests, "should return empty list for non-existent user")
	})
}
