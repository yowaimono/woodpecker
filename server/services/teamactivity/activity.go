// Copyright 2026 Woodpecker Authors
// SPDX-License-Identifier: Apache-2.0

// Package teamactivity collects bounded, read-only GitHub activity for repositories
// already authorized by the caller. It never derives PR counts from CI runs.
package teamactivity

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const MaxRepositories = 8
const MaxPages = 20
const pageSize = 100

type Repository struct {
	ID       int64    `json:"id"`
	Owner    string   `json:"-"`
	Name     string   `json:"-"`
	FullName string   `json:"full_name"`
	Branch   string   `json:"default_branch"`
	Status   string   `json:"status"`
	Issues   []string `json:"issues"`
}

type Event struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	RepoID     int64      `json:"repo_id"`
	Actor      string     `json:"actor"`
	ActorID    string     `json:"actor_id"`
	Avatar     string     `json:"avatar"`
	Bot        bool       `json:"bot"`
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	OccurredAt time.Time  `json:"occurred_at"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}

type Report struct {
	Source       string       `json:"source"`
	Since        time.Time    `json:"since"`
	Until        time.Time    `json:"until"`
	GeneratedAt  time.Time    `json:"generated_at"`
	Partial      bool         `json:"partial"`
	Repositories []Repository `json:"repositories"`
	Events       []Event      `json:"events"`
}

// SelectRepositories only selects from the caller's authorized list. Unknown IDs
// fail closed, and the collection cap is explicit in the returned scope statuses.
func SelectRepositories(repos []Repository, id int64) ([]Repository, error) {
	sort.Slice(repos, func(i, j int) bool { return repos[i].FullName < repos[j].FullName })
	selected := make([]Repository, 0, len(repos))
	for _, repo := range repos {
		if id != 0 && repo.ID != id {
			continue
		}
		repo.Issues = []string{}
		repo.Status = "pending"
		if len(selected) >= MaxRepositories {
			repo.Status = "skipped"
			repo.Issues = append(repo.Issues, "repository_limit")
		}
		selected = append(selected, repo)
	}
	if id != 0 && len(selected) == 0 {
		return nil, errors.New("repository is not in the authorized scope")
	}
	return selected, nil
}

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}

type Collector struct {
	Client  *http.Client
	BaseURL string
	Token   string
}

type actor struct {
	Login  string `json:"login"`
	ID     int64  `json:"id"`
	Avatar string `json:"avatar_url"`
	Type   string `json:"type"`
}
type pullRequest struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	URL       string     `json:"html_url"`
	User      actor      `json:"user"`
	MergedAt  *time.Time `json:"merged_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
type commitIdentity struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}
type commit struct {
	SHA    string `json:"sha"`
	URL    string `json:"html_url"`
	Author *actor `json:"author"`
	Commit struct {
		Message   string         `json:"message"`
		Author    commitIdentity `json:"author"`
		Committer commitIdentity `json:"committer"`
	} `json:"commit"`
}

// Collect returns partial coverage explicitly on provider failures, scan limits or
// cancellation. Identity for unmapped Git authors is SHA-local, never an email.
func (c Collector) Collect(ctx context.Context, repos []Repository, since, until time.Time) Report {
	report := Report{Source: "github", Since: since, Until: until, GeneratedAt: time.Now().UTC(), Repositories: repos, Events: []Event{}}
	if c.Client == nil {
		c.Client = NewHTTPClient()
	}
	batches := make([][]Event, len(report.Repositories))
	var workers sync.WaitGroup
	slots := make(chan struct{}, 3)
	for i := range report.Repositories {
		if report.Repositories[i].Status == "skipped" {
			continue
		}
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				report.Repositories[index].Status = "partial"
				report.Repositories[index].Issues = []string{"collection_cancelled"}
				return
			}
			defer func() { <-slots }()
			repo := &report.Repositories[index]
			events, issues := c.collectRepository(ctx, *repo, since, until)
			repo.Status = "complete"
			repo.Issues = issues
			if len(issues) > 0 {
				repo.Status = "partial"
			}
			batches[index] = events
		}(i)
	}
	workers.Wait()
	report.GeneratedAt = time.Now().UTC()
	seen := map[string]bool{}
	for i, events := range batches {
		if report.Repositories[i].Status != "complete" {
			report.Partial = true
		}
		for _, event := range events {
			if !seen[event.ID] {
				report.Events = append(report.Events, event)
				seen[event.ID] = true
			}
		}
	}
	sort.Slice(report.Events, func(i, j int) bool { return report.Events[i].OccurredAt.After(report.Events[j].OccurredAt) })
	return report
}

func (c Collector) collectRepository(ctx context.Context, repo Repository, since, until time.Time) ([]Event, []string) {
	events := []Event{}
	issues := []string{}
	path := "repos/" + url.PathEscape(repo.Owner) + "/" + url.PathEscape(repo.Name)
	for page := 1; page <= MaxPages; page++ {
		prs := []pullRequest{}
		query := url.Values{"state": {"closed"}, "sort": {"updated"}, "direction": {"desc"}, "per_page": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(page)}}
		more, err := c.get(ctx, path+"/pulls", query, &prs)
		if err != nil {
			issues = append(issues, "pull_requests_"+err.Error())
			break
		}
		reachedSince := false
		for _, pr := range prs {
			if pr.UpdatedAt.Before(since) {
				reachedSince = true
			}
			if pr.MergedAt == nil || pr.MergedAt.Before(since) || !pr.MergedAt.Before(until) {
				continue
			}
			identity := pr.User.Login
			if identity == "" {
				identity = "unknown-pr-author"
			}
			actorID := "github:" + strconv.FormatInt(pr.User.ID, 10)
			if pr.User.ID == 0 {
				actorID = "github-login:" + strings.ToLower(identity)
			}
			created := pr.CreatedAt
			events = append(events, Event{ID: fmt.Sprintf("%d:pr:%d", repo.ID, pr.Number), Kind: "merged_pr", RepoID: repo.ID, Actor: identity, ActorID: actorID, Avatar: pr.User.Avatar, Bot: pr.User.Type == "Bot" || strings.HasSuffix(identity, "[bot]"), Title: pr.Title, URL: pr.URL, OccurredAt: *pr.MergedAt, CreatedAt: &created})
		}
		if !more || len(prs) == 0 || reachedSince {
			break
		}
		if page == MaxPages {
			issues = append(issues, "pull_requests_scan_limit")
		}
	}
	for page := 1; page <= MaxPages; page++ {
		commits := []commit{}
		query := url.Values{"since": {since.Format(time.RFC3339)}, "until": {until.Format(time.RFC3339)}, "per_page": {strconv.Itoa(pageSize)}, "page": {strconv.Itoa(page)}}
		if repo.Branch != "" {
			query.Set("sha", repo.Branch)
		}
		more, err := c.get(ctx, path+"/commits", query, &commits)
		if err != nil {
			issues = append(issues, "commits_"+err.Error())
			break
		}
		for _, commit := range commits {
			// The committer timestamp is also the timestamp used by GitHub's since/until filter.
			occurred := commit.Commit.Committer.Date
			if occurred.Before(since) || !occurred.Before(until) {
				continue
			}
			name := commit.Commit.Author.Name
			identity := commit.Commit.Author.Email
			if identity == "" {
				identity = name
			}
			digest := sha256.Sum256([]byte(strings.ToLower(identity)))
			actorID := fmt.Sprintf("git:%x", digest[:12])
			avatar := ""
			bot := strings.HasSuffix(name, "[bot]")
			if commit.Author != nil && commit.Author.Login != "" {
				name = commit.Author.Login
				actorID = "github:" + strconv.FormatInt(commit.Author.ID, 10)
				avatar = commit.Author.Avatar
				bot = commit.Author.Type == "Bot" || strings.HasSuffix(name, "[bot]")
				if commit.Author.ID == 0 {
					actorID = "github-login:" + strings.ToLower(name)
				}
			}
			if name == "" {
				name = "unknown-author"
				actorID = "git:" + commit.SHA
			}
			events = append(events, Event{ID: fmt.Sprintf("%d:commit:%s", repo.ID, commit.SHA), Kind: "commit", RepoID: repo.ID, Actor: name, ActorID: actorID, Avatar: avatar, Bot: bot, Title: strings.SplitN(commit.Commit.Message, "\n", 2)[0], URL: commit.URL, OccurredAt: occurred})
		}
		if !more || len(commits) == 0 {
			break
		}
		if page == MaxPages {
			issues = append(issues, "commits_scan_limit")
		}
	}
	return events, issues
}

// get emits stable error codes, never raw provider errors, tokens, or response bodies.
func (c Collector) get(ctx context.Context, path string, query url.Values, dest any) (bool, error) {
	endpoint := strings.TrimRight(c.BaseURL, "/") + "/" + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, errors.New("request_failed")
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Woodpecker-Team-Activity")
	client := c.Client
	if client == nil {
		client = NewHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, errors.New("unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("http_%d", resp.StatusCode)
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(dest); err != nil {
		return false, errors.New("invalid_response")
	}
	return strings.Contains(resp.Header.Get("Link"), `rel="next"`), nil
}
