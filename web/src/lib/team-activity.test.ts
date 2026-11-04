import { describe, expect, it } from 'vitest';

import type { TeamActivityEvent, TeamActivityReport } from '~/lib/api/types/teamActivity';

import { activityEvents, csvCell, localDate, median, rankMembers, summarizeActivity } from './team-activity';

const dataEnd = '2026-11-03T00:00:00Z';
const event = (overrides: Partial<TeamActivityEvent> = {}): TeamActivityEvent => ({
  id: 'commit:1',
  kind: 'commit',
  repo_id: 1,
  actor: 'alice',
  actor_id: 'github:1',
  avatar: '',
  bot: false,
  title: 'Improve deployment',
  url: 'https://example.test/commit/1',
  occurred_at: '2026-11-01T16:30:00Z',
  ...overrides,
});
const report = (events: TeamActivityEvent[]): TeamActivityReport => ({
  source: 'github',
  since: '2026-10-27T00:00:00Z',
  until: '2026-11-03T00:00:00Z',
  generated_at: '2026-11-03T00:00:00Z',
  partial: false,
  repositories: [{ id: 1, full_name: 'team/backend', default_branch: 'main', status: 'complete', issues: [] }],
  events,
});

describe('team activity statistics', () => {
  it('deduplicates events and excludes bots and both outside-window boundaries', () => {
    const data = report([
      event(),
      event(),
      event({ id: 'bot', bot: true }),
      event({ id: 'end', occurred_at: dataEnd }),
      event({ id: 'old', occurred_at: '2026-10-26T23:59:59Z' }),
      event({ id: 'bad', occurred_at: 'invalid' }),
    ]);
    expect(activityEvents(data, true)).toHaveLength(1);
    expect(activityEvents(data, false)).toHaveLength(2);
  });

  it('groups active days in the chosen timezone without double-counting PRs and commits as days', () => {
    const data = report([
      event(),
      event({ id: 'second', occurred_at: '2026-11-02T01:00:00Z' }),
      event({ id: 'pr', kind: 'merged_pr', created_at: '2026-11-01T01:00:00Z', occurred_at: '2026-11-02T01:00:00Z' }),
    ]);
    const shanghai = summarizeActivity(data, 'Asia/Shanghai', true);
    const utc = summarizeActivity(data, 'UTC', true);
    expect(shanghai.activeDays).toBe(1);
    expect(utc.activeDays).toBe(2);
    expect(shanghai.commits).toBe(2);
    expect(shanghai.merged).toBe(1);
    expect(shanghai.medianMergeHours).toBe(24);
    expect(shanghai.heatmap.flat().reduce((sum, value) => sum + value, 0)).toBe(2);
    expect(shanghai.heatmap[0][0]).toBe(1);
    expect(shanghai.daily.find((day) => day.date === '2026-11-02')).toEqual({
      date: '2026-11-02',
      commits: 2,
      merged: 1,
    });
  });

  it('uses stable identities and keeps different actors with identical names separate', () => {
    const data = summarizeActivity(
      report([event(), event({ id: 'different', actor_id: 'git:other', actor: 'alice' })]),
      'UTC',
      true,
    );
    expect(data.members).toHaveLength(2);
  });

  it('uses competition ranking for ties and changes the metric without changing statistics', () => {
    const data = summarizeActivity(
      report([
        event(),
        event({ id: 'second', actor: 'bob', actor_id: 'github:2' }),
        event({ id: 'pr', actor: 'bob', actor_id: 'github:2', kind: 'merged_pr' }),
      ]),
      'UTC',
      true,
    );
    expect(rankMembers(data.members, 'commits').map((member) => member.rank)).toEqual([1, 1]);
    expect(rankMembers(data.members, 'merged')[0].login).toBe('bob');
  });

  it('handles empty data and invalid or missing PR durations without inventing a zero median', () => {
    expect(summarizeActivity(report([]), 'UTC', true).medianMergeHours).toBeNull();
    const data = summarizeActivity(
      report([event({ kind: 'merged_pr', created_at: '2026-11-02T00:00:00Z' })]),
      'UTC',
      true,
    );
    expect(data.medianMergeHours).toBeNull();
    expect(median([3, 1])).toBe(2);
  });

  it('ranks repositories by deduplicated visible events and counts distinct members and local days', () => {
    const data = report([
      event(),
      event(),
      event({ id: 'pr', kind: 'merged_pr' }),
      event({ id: 'second-author', actor_id: 'github:2', actor: 'bob' }),
      event({ id: 'second-repo', repo_id: 2 }),
      event({ id: 'excluded-bot', repo_id: 2, bot: true }),
    ]);
    data.repositories.push({
      id: 2,
      full_name: 'team/frontend',
      default_branch: 'main',
      status: 'complete',
      issues: [],
    });
    const result = summarizeActivity(data, 'Asia/Shanghai', true).repositoryActivity;
    expect(result.map((repo) => repo.id)).toEqual([1, 2]);
    expect(result[0]).toMatchObject({ commits: 2, merged: 1, members: 2, activeDays: 1, total: 3, share: 0.75 });
    expect(result[1]).toMatchObject({ commits: 1, merged: 0, members: 1, total: 1, share: 0.25 });
    expect(summarizeActivity(data, 'UTC', false).repositoryActivity[1].total).toBe(2);
  });

  it('retains repositories with missing coverage and handles no events without NaN shares', () => {
    const data = report([]);
    data.partial = true;
    data.repositories[0].status = 'partial';
    data.repositories[0].issues = ['commits_http_403'];
    const [repo] = summarizeActivity(data, 'UTC', true).repositoryActivity;
    expect(repo).toMatchObject({ status: 'partial', total: 0, share: 0 });
    expect(repo.issues).toEqual(['commits_http_403']);
  });

  it('handles DST by formatting each event in the selected timezone', () => {
    expect(localDate('2026-11-01T08:30:00Z', 'America/Los_Angeles')).toBe('2026-11-01');
    expect(localDate('2026-11-01T09:30:00Z', 'America/Los_Angeles')).toBe('2026-11-01');
  });

  it.each(['=HYPERLINK("url")', '+cmd', '-cmd', '@name', '\t=cmd', ' =cmd'])(
    'escapes spreadsheet formulas in CSV: %s',
    (value) => {
      expect(csvCell(value).startsWith('"\'')).toBe(true);
    },
  );

  it('quotes commas and embedded double quotes in CSV', () => {
    expect(csvCell('alice,"ops"')).toBe('"alice,""ops"""');
    expect(csvCell(42)).toBe('"42"');
  });
});
