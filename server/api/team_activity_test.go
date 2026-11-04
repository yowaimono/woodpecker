// Copyright 2026 Woodpecker Authors
// SPDX-License-Identifier: Apache-2.0
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.woodpecker-ci.org/woodpecker/v3/server"
	forge_mocks "go.woodpecker-ci.org/woodpecker/v3/server/forge/mocks"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	manager_mocks "go.woodpecker-ci.org/woodpecker/v3/server/services/mocks"
	store_mocks "go.woodpecker-ci.org/woodpecker/v3/server/store/mocks"
)

func teamActivityContext(t *testing.T, query string) (*gin.Context, *httptest.ResponseRecorder, *model.User, *store_mocks.MockStore, *forge_mocks.MockForge) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/team-activity"+query, nil)
	user := &model.User{ID: 17, ForgeID: 2, Login: "alice", AccessToken: "private-token"}
	c.Set("user", user)
	s := store_mocks.NewMockStore(t)
	c.Set("store", s)
	previous := server.Config.Services.Manager
	t.Cleanup(func() { server.Config.Services.Manager = previous })
	manager := manager_mocks.NewMockManager(t)
	provider := forge_mocks.NewMockForge(t)
	manager.On("ForgeFromUser", user).Return(provider, nil)
	server.Config.Services.Manager = manager
	provider.On("Name").Return("github")
	return c, rec, user, s, provider
}

func TestTeamActivityRejectsUnauthenticatedAndInvalidInputs(t *testing.T) {
	for _, query := range []string{"?days=0", "?days=-1", "?days=365", "?days=oops", "?repo_id=-5", "?repo_id=oops"} {
		t.Run(query, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/user/team-activity"+query, nil)
			c.Set("user", &model.User{ID: 1})
			GetTeamActivity(c)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/team-activity", nil)
	GetTeamActivity(c)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTeamActivityCannotReadUnrelatedRepository(t *testing.T) {
	c, rec, user, s, _ := teamActivityContext(t, "?repo_id=99")
	s.On("RepoList", user, true, true, mock.Anything).Return([]*model.Repo{{ID: 1, ForgeID: 2, Owner: "team", Name: "allowed", FullName: "team/allowed"}}, nil)
	GetTeamActivity(c)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTeamActivityExcludesOtherForgeAndReturnsEmptyScope(t *testing.T) {
	c, rec, user, s, provider := teamActivityContext(t, "?days=7")
	s.On("RepoList", user, true, true, mock.Anything).Return([]*model.Repo{{ID: 1, ForgeID: 9, Owner: "other", Name: "private", FullName: "other/private"}}, nil)
	provider.On("URL").Return("https://github.com")
	GetTeamActivity(c)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"repositories":[]`)
	require.Contains(t, rec.Body.String(), `"events":[]`)
	require.NotContains(t, rec.Body.String(), "private-token")
	require.NotContains(t, rec.Body.String(), "other/private")
	require.Equal(t, "private, no-store", rec.Header().Get("Cache-Control"))
}

func TestTeamActivityCollectsAuthorizedScopeWithCallerToken(t *testing.T) {
	c, rec, user, s, provider := teamActivityContext(t, "?days=7&repo_id=1")
	s.On("RepoList", user, true, true, mock.Anything).Return([]*model.Repo{{ID: 1, ForgeID: 2, Owner: "team", Name: "allowed", FullName: "team/allowed", Branch: "main"}}, nil)
	now := time.Now().UTC().Add(-time.Hour)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer private-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v3/repos/team/allowed/pulls":
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"number": 42, "title": "A merged PR", "created_at": now.Add(-time.Hour), "merged_at": now, "updated_at": now, "user": map[string]any{"id": 7, "login": "contributor"}}})
		case "/api/v3/repos/team/allowed/commits":
			require.Equal(t, "main", r.URL.Query().Get("sha"))
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"sha": "abc123", "author": map[string]any{"id": 7, "login": "contributor"}, "commit": map[string]any{"message": "Improve API", "author": map[string]any{"name": "Private Name", "email": "private@example.test"}, "committer": map[string]any{"date": now}}}})
		default:
			t.Errorf("out-of-scope request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	provider.On("URL").Return(upstream.URL)
	GetTeamActivity(c)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"partial":false`)
	require.Contains(t, rec.Body.String(), `"kind":"merged_pr"`)
	require.Contains(t, rec.Body.String(), `"kind":"commit"`)
	require.Contains(t, rec.Body.String(), `"actor":"contributor"`)
	require.NotContains(t, rec.Body.String(), "private@example.test")
	require.NotContains(t, rec.Body.String(), "private-token")
}
