// Copyright 2026 Woodpecker Authors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go.woodpecker-ci.org/woodpecker/v3/server"
	"go.woodpecker-ci.org/woodpecker/v3/server/forge"
	"go.woodpecker-ci.org/woodpecker/v3/server/router/middleware/session"
	"go.woodpecker-ci.org/woodpecker/v3/server/services/teamactivity"
	"go.woodpecker-ci.org/woodpecker/v3/server/store"
)

// GetTeamActivity reads GitHub activity only for the authenticated user's active
// repositories. No machine credentials, private emails, or cross-user cache are used.
func GetTeamActivity(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	user := session.User(c)
	if user == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	days := 30
	if value := c.Query("days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || (parsed != 7 && parsed != 30 && parsed != 90) {
			c.String(http.StatusBadRequest, "days must be 7, 30 or 90")
			return
		}
		days = parsed
	}
	repoID := int64(0)
	if value := c.Query("repo_id"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			c.String(http.StatusBadRequest, "invalid repo_id")
			return
		}
		repoID = parsed
	}
	provider, err := server.Config.Services.Manager.ForgeFromUser(user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if provider.Name() != "github" {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "unsupported_forge"})
		return
	}
	s := store.FromContext(c)
	repos, err := s.RepoList(user, true, true, nil)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	authorized := make([]teamactivity.Repository, 0, len(repos))
	for _, repo := range repos {
		if repo.ForgeID != user.ForgeID {
			continue
		}
		authorized = append(authorized, teamactivity.Repository{ID: repo.ID, Owner: repo.Owner, Name: repo.Name, FullName: repo.FullName, Branch: repo.Branch})
	}
	selected, err := teamactivity.SelectRepositories(authorized, repoID)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	forge.Refresh(ctx, provider, s, user)
	endpoint := "https://api.github.com"
	if host := strings.TrimRight(provider.Config().URL, "/"); host != "https://github.com" {
		endpoint = host + "/api/v3"
	}
	until := time.Now().UTC()
	collector := teamactivity.Collector{BaseURL: endpoint, Token: user.AccessToken}
	report := collector.Collect(ctx, selected, until.Add(-time.Duration(days)*24*time.Hour), until)
	c.JSON(http.StatusOK, report)
}
