import type { TeamActivityEvent, TeamActivityReport } from '~/lib/api/types/teamActivity';

export interface ActivityMember {
  id: string;
  login: string;
  avatar: string;
  bot: boolean;
  commits: number;
  merged: number;
  activeDays: number;
  repositories: number;
  lastActive: string;
  medianMergeHours: number | null;
}

export function localDate(timestamp: string, timeZone: string): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date(timestamp));
  const value = (name: string) => parts.find((part) => part.type === name)?.value ?? '';
  return `${value('year')}-${value('month')}-${value('day')}`;
}

export function median(values: number[]): number | null {
  if (values.length === 0) return null;
  const sorted = [...values].sort((a, b) => a - b);
  const center = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 1 ? sorted[center] : (sorted[center - 1] + sorted[center]) / 2;
}

export function mergeHours(event: TeamActivityEvent): number | null {
  if (event.kind !== 'merged_pr' || event.created_at === undefined || event.created_at === '') return null;
  const hours = (Date.parse(event.occurred_at) - Date.parse(event.created_at)) / 3_600_000;
  return Number.isFinite(hours) && hours >= 0 ? hours : null;
}

/** Duplicate webhook/provider pages cannot inflate counts. Only the report's half-open window is included. */
export function activityEvents(report: TeamActivityReport, excludeBots: boolean): TeamActivityEvent[] {
  const seen = new Set<string>();
  const since = Date.parse(report.since);
  const until = Date.parse(report.until);
  return report.events.filter((event) => {
    const time = Date.parse(event.occurred_at);
    if (seen.has(event.id) || !Number.isFinite(time) || time < since || time >= until || (excludeBots && event.bot))
      return false;
    seen.add(event.id);
    return true;
  });
}

export function summarizeActivity(report: TeamActivityReport, timeZone: string, excludeBots: boolean) {
  const events = activityEvents(report, excludeBots);
  const grouped = new Map<string, TeamActivityEvent[]>();
  events.forEach((event) => {
    const group = grouped.get(event.actor_id);
    if (group) group.push(event);
    else grouped.set(event.actor_id, [event]);
  });
  const members: ActivityMember[] = [...grouped.entries()].map(([id, items]) => ({
    id,
    login: items[0].actor,
    avatar: items[0].avatar,
    bot: items.some((item) => item.bot),
    commits: items.filter((item) => item.kind === 'commit').length,
    merged: items.filter((item) => item.kind === 'merged_pr').length,
    activeDays: new Set(items.map((item) => localDate(item.occurred_at, timeZone))).size,
    repositories: new Set(items.map((item) => item.repo_id)).size,
    lastActive: items.reduce(
      (latest, item) => (Date.parse(item.occurred_at) > Date.parse(latest) ? item.occurred_at : latest),
      items[0].occurred_at,
    ),
    medianMergeHours: median(items.map(mergeHours).filter((value): value is number => value !== null)),
  }));
  const firstDate = localDate(report.since, timeZone);
  const lastDate = localDate(report.until, timeZone);
  const daily: { date: string; commits: number; merged: number }[] = [];
  for (
    let cursor = new Date(`${firstDate}T12:00:00Z`);
    cursor.toISOString().slice(0, 10) <= lastDate;
    cursor.setUTCDate(cursor.getUTCDate() + 1)
  ) {
    daily.push({ date: cursor.toISOString().slice(0, 10), commits: 0, merged: 0 });
  }
  const dailyMap = new Map(daily.map((day) => [day.date, day]));
  // Monday first, UTC pseudo-dates only determine weekday after local-date conversion.
  const heatmap = Array.from({ length: 7 }, () => Array.from<number>({ length: 24 }).fill(0));
  const hourFormatter = new Intl.DateTimeFormat('en-GB', { timeZone, hour: 'numeric', hourCycle: 'h23' });
  events.forEach((event) => {
    const date = localDate(event.occurred_at, timeZone);
    const day = dailyMap.get(date);
    if (day) day[event.kind === 'commit' ? 'commits' : 'merged']++;
    if (event.kind === 'commit') {
      const weekday = (new Date(`${date}T12:00:00Z`).getUTCDay() + 6) % 7;
      const hour = Number(hourFormatter.format(new Date(event.occurred_at)));
      heatmap[weekday][hour]++;
    }
  });
  return {
    events,
    members,
    daily,
    heatmap,
    commits: events.filter((event) => event.kind === 'commit').length,
    merged: events.filter((event) => event.kind === 'merged_pr').length,
    activeDays: new Set(events.map((event) => localDate(event.occurred_at, timeZone))).size,
    activeRepositories: new Set(events.map((event) => event.repo_id)).size,
    medianMergeHours: median(events.map(mergeHours).filter((value): value is number => value !== null)),
  };
}

// Competition ranks reflect ties (1, 1, 3), with deterministic alphabetical ordering within a tie.
export function rankMembers(members: ActivityMember[], metric: 'merged' | 'commits' | 'activeDays') {
  const sorted = [...members].sort((a, b) => b[metric] - a[metric] || a.login.localeCompare(b.login));
  return sorted.map((member, index) => ({
    ...member,
    rank: index === 0 ? 1 : sorted.findIndex((item) => item[metric] === member[metric]) + 1,
  }));
}

/** Excel-safe CSV cells protect exported user-controlled names from formula execution. */
export function csvCell(value: string | number): string {
  const text = String(value);
  const safe = /^\s*[=+@-]|^[\t\r\n]/.test(text) ? `'${text}` : text;
  return `"${safe.replaceAll('"', '""')}"`;
}
