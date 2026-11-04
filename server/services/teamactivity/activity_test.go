// Copyright 2026 Woodpecker Authors
// SPDX-License-Identifier: Apache-2.0
package teamactivity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testWindow() (time.Time, time.Time) {
	return time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
}
func testRepo() Repository {
	return Repository{ID: 1, Owner: "acme", Name: "backend", FullName: "acme/backend", Branch: "main", Status: "pending", Issues: []string{}}
}
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func TestCollectCountsOnlyMergedPRsAndWindowedCommits(t *testing.T) {
	since, until := testWindow()
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer user-token" {
			t.Error("must use read-only calls with the requesting user's token")
		}
		if strings.HasSuffix(r.URL.Path, "/pulls") {
			if r.URL.Query().Get("state") != "closed" || r.URL.Query().Get("sort") != "updated" {
				t.Error("wrong PR scan ordering")
			}
			writeJSON(w, []any{
				map[string]any{"number": 1, "title": "Merged", "html_url": "https://example.test/pr/1", "user": map[string]any{"login": "alice", "id": 12}, "created_at": "2026-10-31T12:00:00Z", "updated_at": "2026-11-01T12:00:00Z", "merged_at": "2026-11-01T12:00:00Z"},
				map[string]any{"number": 2, "title": "Closed only", "updated_at": "2026-11-02T12:00:00Z", "merged_at": nil},
				map[string]any{"number": 3, "title": "Old merge, recently updated", "updated_at": "2026-11-02T12:00:00Z", "merged_at": "2026-09-01T12:00:00Z"},
			})
			return
		}
		if r.URL.Query().Get("sha") != "main" || r.URL.Query().Get("since") != since.Format(time.RFC3339) {
			t.Error("commit scope must match the default branch and time window")
		}
		writeJSON(w, []any{
			map[string]any{"sha": "abc", "author": map[string]any{"id": 12, "login": "alice"}, "commit": map[string]any{"message": "Improve API\nDetails", "author": map[string]any{"name": "Alice", "email": "private@example.test"}, "committer": map[string]any{"date": "2026-11-02T23:00:00Z"}}},
			map[string]any{"sha": "future", "commit": map[string]any{"committer": map[string]any{"date": until.Format(time.RFC3339)}}},
		})
	}))
	defer provider.Close()
	report := (Collector{BaseURL: provider.URL, Token: "user-token"}).Collect(context.Background(), []Repository{testRepo()}, since, until)
	if report.Partial || len(report.Events) != 2 || calls != 2 {
		t.Fatalf("wrong report: %+v", report)
	}
	if report.Events[0].Kind != "commit" || report.Events[0].Title != "Improve API" || report.Events[0].ActorID != report.Events[1].ActorID {
		t.Fatalf("bad attribution: %+v", report.Events)
	}
	data, _ := json.Marshal(report)
	if strings.Contains(string(data), "private@example.test") || strings.Contains(string(data), "user-token") {
		t.Fatal("private data leaked")
	}
}

func TestPaginationDeduplicationAndTruncation(t *testing.T) {
	since, until := testWindow()
	pullCalls, commitCalls := 0, 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<https://unused.example/page>; rel="next"`)
		if strings.HasSuffix(r.URL.Path, "/pulls") {
			pullCalls++
			writeJSON(w, []any{map[string]any{"number": pullCalls, "title": "PR", "updated_at": "2026-11-01T00:00:00Z", "merged_at": "2026-11-01T00:00:00Z", "created_at": "2026-10-31T00:00:00Z"}})
			return
		}
		commitCalls++
		writeJSON(w, []any{map[string]any{"sha": "same-sha", "commit": map[string]any{"author": map[string]any{"name": "A"}, "committer": map[string]any{"date": "2026-11-01T00:00:00Z"}}}})
	}))
	defer provider.Close()
	report := (Collector{BaseURL: provider.URL}).Collect(context.Background(), []Repository{testRepo()}, since, until)
	if pullCalls != MaxPages || commitCalls != MaxPages || !report.Partial || len(report.Events) != MaxPages+1 {
		t.Fatalf("pagination must be bounded and deduplicated: %+v", report)
	}
	if strings.Join(report.Repositories[0].Issues, ",") != "pull_requests_scan_limit,commits_scan_limit" {
		t.Fatal(report.Repositories[0].Issues)
	}
}

func TestFailuresArePartialAndNeverLeakUpstreamBodies(t *testing.T) {
	since, until := testWindow()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
		_, _ = fmt.Fprint(w, "secret response body")
	}))
	defer provider.Close()
	report := (Collector{BaseURL: provider.URL}).Collect(context.Background(), []Repository{testRepo()}, since, until)
	if !report.Partial || report.Repositories[0].Status != "partial" || len(report.Events) != 0 {
		t.Fatal("failure presented as complete")
	}
	data, _ := json.Marshal(report)
	if strings.Contains(string(data), "secret response body") {
		t.Fatal("provider body leaked")
	}
	if !strings.Contains(string(data), "http_403") {
		t.Fatal("stable reason code missing")
	}
}

func TestPRScanStopsAfterUpdatedCutoff(t *testing.T) {
	since, until := testWindow()
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pulls") {
			calls++
			w.Header().Set("Link", `<https://unused.example/page>; rel="next"`)
			writeJSON(w, []any{map[string]any{"number": 1, "updated_at": "2026-09-01T00:00:00Z", "merged_at": "2026-09-01T00:00:00Z"}})
			return
		}
		writeJSON(w, []any{})
	}))
	defer provider.Close()
	report := (Collector{BaseURL: provider.URL}).Collect(context.Background(), []Repository{testRepo()}, since, until)
	if calls != 1 || report.Partial {
		t.Fatalf("PR cutoff should end a complete scan: %+v", report)
	}
}

func TestSelectRepositoriesFailsClosedAndDisclosesLimit(t *testing.T) {
	repos := []Repository{}
	for i := int64(1); i <= MaxRepositories+2; i++ {
		repos = append(repos, Repository{ID: i, FullName: fmt.Sprintf("team/repo-%02d", i)})
	}
	if _, err := SelectRepositories(repos, 999); err == nil {
		t.Fatal("unauthorized repository accepted")
	}
	selected, err := SelectRepositories(repos, 0)
	if err != nil || len(selected) != MaxRepositories+2 || selected[MaxRepositories].Status != "skipped" {
		t.Fatal("scope truncation hidden")
	}
	selected, err = SelectRepositories(repos, MaxRepositories+2)
	if err != nil || len(selected) != 1 || selected[0].Status != "pending" {
		t.Fatal("explicit selection must bypass the collection position cap")
	}
}

func TestCancelledRequestIsNotComplete(t *testing.T) {
	since, until := testWindow()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := (Collector{BaseURL: "http://127.0.0.1:1"}).Collect(ctx, []Repository{testRepo()}, since, until)
	if !report.Partial {
		t.Fatal("cancellation silently became zero activity")
	}
}

func TestClientDoesNotFollowCredentialBearingRedirects(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true; writeJSON(w, []any{}) }))
	defer target.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer provider.Close()
	since, until := testWindow()
	report := (Collector{BaseURL: provider.URL, Token: "secret"}).Collect(context.Background(), []Repository{testRepo()}, since, until)
	if called || !report.Partial {
		t.Fatal("redirect followed with credentials")
	}
}
