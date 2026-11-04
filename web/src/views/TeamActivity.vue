<template>
  <div class="team-page">
    <header class="team-header">
      <div>
        <div class="team-eyebrow">{{ t('team_activity.eyebrow') }}</div>
        <h1 v-text="t('team_activity.title')" />
        <p v-text="t('team_activity.subtitle')" />
      </div>
      <Button
        start-icon="refresh"
        :text="t('team_activity.refresh')"
        :is-loading="loading"
        :disabled="loading"
        @click="load"
      />
    </header>
    <section class="team-toolbar" :aria-label="t('team_activity.filters')">
      <div class="period-switch" role="group" :aria-label="t('team_activity.period')">
        <button
          v-for="period in periods"
          :key="period"
          type="button"
          :aria-pressed="days === period"
          @click="days = period"
          v-text="t('team_activity.days', { count: period })"
        />
      </div>
      <!-- prettier-ignore -->
      <label class="toolbar-select"><Icon name="repo" /><span class="sr-only" v-text="t('team_activity.repository')" />
        <select v-model="repoId" :aria-label="t('team_activity.repository')">
          <option value="" v-text="t('team_activity.all_repositories')" />
          <option v-for="repo in repositories" :key="repo.id" :value="String(repo.id)" v-text="repo.full_name" />
          <!-- End of options -->
        </select>
      </label>
      <!-- prettier-ignore -->
      <label class="toolbar-select"><Icon name="since" /><span class="sr-only" v-text="t('team_activity.timezone')" />
        <select v-model="timeZone" :aria-label="t('team_activity.timezone')">
          <option v-for="zone in timeZones" :key="zone" :value="zone" v-text="zone" />
          <!-- End of options -->
        </select>
      </label>
      <!-- prettier-ignore -->
      <label class="bot-toggle"><input v-model="excludeBots" type="checkbox" />{{ t('team_activity.exclude_bots') }}</label>
    </section>
    <div v-if="loading" class="team-state" role="status" aria-live="polite">
      <Icon name="spinner" />
      <h2 v-text="t('team_activity.loading')" />
      <p v-text="t('team_activity.loading_detail')" />
    </div>
    <div v-else-if="errorMessage" class="team-state" role="alert">
      <Icon name="alert" />
      <h2 v-text="errorMessage" />
      <p v-text="t('team_activity.error_detail')" />
      <Button :text="t('team_activity.retry')" color="blue" @click="load" />
    </div>
    <template v-else-if="report && summary">
      <div class="scope-line">
        <span class="source-badge" v-text="t('team_activity.github_source')" /><span
          v-text="t('team_activity.scope', { count: collectedRepositories, total: report.repositories.length })"
        /><span class="scope-updated" v-text="t('team_activity.updated', { time: formatDate(report.generated_at) })" />
      </div>
      <div v-if="report.partial" class="coverage-warning" role="status">
        <Icon name="alert" />
        <div>
          <strong v-text="t('team_activity.partial')" />
          <p v-text="t('team_activity.partial_detail')" />
        </div>
      </div>
      <section class="team-metrics" :aria-label="t('team_activity.overview')">
        <article
          v-for="metric in metrics"
          :key="metric.key"
          class="metric-card"
          :class="{ 'metric-featured': metric.key === 'merged' }"
        >
          <div class="metric-label"><span v-text="metric.label" /><Icon :name="metric.icon" /></div>
          <strong v-text="metric.value" /><span class="metric-footnote" v-text="metric.detail" />
        </article>
      </section>
      <div class="team-main-grid">
        <section class="team-panel trend-panel" :aria-label="t('team_activity.trend')">
          <div class="panel-heading">
            <div>
              <h2 v-text="t('team_activity.trend')" />
              <p v-text="t('team_activity.trend_detail')" />
            </div>
            <!-- prettier-ignore -->
            <div class="chart-legend">
              <span><i class="legend-commit" />{{ t('team_activity.commits') }}</span><span><i class="legend-merge" />{{ t('team_activity.merged') }}</span>
            </div>
          </div>
          <div
            v-if="summary.events.length"
            class="daily-chart"
            role="img"
            :aria-label="t('team_activity.trend_accessible', { commits: summary.commits, merged: summary.merged })"
          >
            <div class="chart-scale">
              <span v-text="trendMax" /><span v-text="Math.round(trendMax / 2)" /><span v-text="number(0)" />
            </div>
            <div class="daily-columns">
              <div
                v-for="day in summary.daily"
                :key="day.date"
                class="daily-column"
                :title="t('team_activity.day_counts', { date: day.date, commits: day.commits, merged: day.merged })"
              >
                <div class="daily-bar" :style="{ height: `${((day.commits + day.merged) / trendMax) * 100}%` }">
                  <span class="bar-merge" :style="{ flexGrow: day.merged }" /><span
                    class="bar-commit"
                    :style="{ flexGrow: day.commits }"
                  />
                </div>
              </div>
            </div>
          </div>
          <div v-else class="chart-empty">{{ t('team_activity.no_records') }}</div>
          <div class="chart-axis">
            <span v-text="shortDate(report.since)" /><span v-text="shortDate(report.until)" />
          </div>
        </section>
        <section class="team-panel rhythm-panel" :aria-label="t('team_activity.rhythm')">
          <div class="panel-heading">
            <div>
              <h2 v-text="t('team_activity.rhythm')" />
              <p v-text="timeZone" />
            </div>
            <Icon name="since" />
          </div>
          <div class="heatmap-scroll">
            <div class="heatmap" role="img" :aria-label="t('team_activity.heatmap_accessible')">
              <div class="heatmap-hours">
                <span
                  v-for="hour in [0, 6, 12, 18, 23]"
                  :key="hour"
                  :style="{ gridColumn: `${hour + 1} / span 1` }"
                  v-text="String(hour).padStart(2, '0')"
                />
              </div>
              <div v-for="(hourCounts, weekday) in summary.heatmap" :key="weekday" class="heatmap-row">
                <span class="weekday-label" v-text="weekdays[weekday]" /><span
                  v-for="(count, hour) in hourCounts"
                  :key="hour"
                  class="heatmap-cell"
                  :data-level="heatLevel(count)"
                  :title="t('team_activity.hour_count', { day: weekdays[weekday], hour, count })"
                />
              </div>
            </div>
          </div>
          <div class="heatmap-legend">
            <span v-text="t('team_activity.less')" /><i
              v-for="level in [0, 1, 2, 3, 4]"
              :key="level"
              class="heatmap-cell"
              :data-level="level"
            /><span v-text="t('team_activity.more')" />
          </div>
          <p class="rhythm-note" v-text="t('team_activity.time_note')" />
        </section>
      </div>
      <section class="team-panel leaderboard-panel" :aria-label="t('team_activity.leaderboard')">
        <div class="panel-heading leaderboard-heading">
          <div>
            <h2>
              {{ t('team_activity.leaderboard') }}<span class="heading-count" v-text="number(summary.members.length)" />
            </h2>
            <p v-text="t('team_activity.ranking_note')" />
          </div>
          <!-- prettier-ignore -->
          <div class="ranking-actions">
            <TextField
              v-model="memberSearch"
              :placeholder="t('team_activity.search_member')"
              :aria-label="t('team_activity.search_member')"
              class="member-search"
            /><!-- prettier-ignore -->
            <select v-model="rankBy" class="rank-select" :aria-label="t('team_activity.sort')">
              <option value="merged" v-text="t('team_activity.sort_merged')" />
              <option value="commits" v-text="t('team_activity.sort_commits')" />
              <option value="activeDays" v-text="t('team_activity.sort_days')" />
              <!-- End of options -->
</select><Button
              start-icon="download"
              :text="t('team_activity.export')"
              :disabled="rankedMembers.length === 0"
              @click="exportCSV"
            />
          </div>
        </div>
        <div class="leaderboard-scroll">
          <table class="leaderboard-table">
            <caption class="sr-only">
              {{
                t('team_activity.leaderboard')
              }}
            </caption>
            <thead>
              <tr>
                <th scope="col" v-text="t('team_activity.rank')" />
                <th scope="col" v-text="t('team_activity.member')" />
                <th scope="col" v-text="t('team_activity.merged')" />
                <th scope="col" v-text="t('team_activity.commits')" />
                <th scope="col" v-text="t('team_activity.active_days')" />
                <th scope="col" v-text="t('team_activity.repositories')" />
                <th scope="col" v-text="t('team_activity.merge_time')" />
                <th scope="col" v-text="t('team_activity.last_active')" />
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in rankedMembers" :key="member.id">
                <td>
                  <span class="rank-number" :class="{ 'rank-top': member.rank <= 3 && member[rankBy] > 0 }">{{
                    number(member.rank)
                  }}</span>
                </td>
                <th scope="row">
                  <div class="member-cell">
                    <span
                      class="member-avatar"
                      :style="{ '--avatar-hue': `${avatarHue(member.login)}deg` }"
                      v-text="member.login.slice(0, 2).toUpperCase()"
                    />
                    <div>
                      <span class="member-login" v-text="member.login" /><small
                        v-if="member.bot"
                        v-text="t('team_activity.bot')"
                      /><small v-else-if="member.id.startsWith('git:')" v-text="t('team_activity.unlinked_author')" />
                    </div>
                  </div>
                </th>
                <td :class="{ 'sorted-value': rankBy === 'merged' }" v-text="number(member.merged)" />
                <td :class="{ 'sorted-value': rankBy === 'commits' }" v-text="number(member.commits)" />
                <td :class="{ 'sorted-value': rankBy === 'activeDays' }" v-text="number(member.activeDays)" />
                <td v-text="number(member.repositories)" />
                <td v-text="hours(member.medianMergeHours)" />
                <td class="last-active" v-text="formatDate(member.lastActive)" />
              </tr>
              <tr v-if="rankedMembers.length === 0">
                <td
                  colspan="8"
                  class="empty-table"
                  v-text="memberSearch ? t('team_activity.no_members') : t('team_activity.no_records')"
                />
              </tr>
            </tbody>
          </table>
        </div>
      </section>
      <div class="team-bottom-grid">
        <section class="team-panel recent-panel" :aria-label="t('team_activity.recent')">
          <div class="panel-heading">
            <div>
              <h2 v-text="t('team_activity.recent')" />
              <p v-text="t('team_activity.recent_detail')" />
            </div>
          </div>
          <ul v-if="recentEvents.length" class="recent-list">
            <li v-for="event in recentEvents" :key="event.id">
              <!-- prettier-ignore -->
              <span class="event-icon" :class="{ 'event-merged': event.kind === 'merged_pr' }"><Icon :name="event.kind === 'merged_pr' ? 'pull-request-closed' : 'commit'" /></span>
              <div class="event-content">
                <div class="event-top">
                  <span
                    class="event-kind"
                    v-text="event.kind === 'merged_pr' ? t('team_activity.merged_pr') : t('team_activity.commit')"
                  /><time :datetime="event.occurred_at" v-text="formatDate(event.occurred_at)" />
                </div>
                <a
                  v-if="safeURL(event.url)"
                  :href="safeURL(event.url)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="event-title"
                  v-text="event.title"
                /><span v-else class="event-title" v-text="event.title" />
                <p>
                  {{ event.actor }}<span class="event-separator" aria-hidden="true" />{{
                    repositoryName(event.repo_id)
                  }}
                </p>
              </div>
            </li>
          </ul>
          <div v-else class="chart-empty">{{ t('team_activity.no_records') }}</div>
          <button
            v-if="summary.events.length > recentLimit"
            class="more-events"
            type="button"
            @click="recentLimit += 8"
          >
            {{ t('team_activity.load_more') }}
          </button>
        </section>
        <section class="team-panel scope-panel" :aria-label="t('team_activity.methodology')">
          <div class="panel-heading">
            <div>
              <h2 v-text="t('team_activity.methodology')" />
              <p v-text="t('team_activity.rolling_window')" />
            </div>
            <Icon name="info" />
          </div>
          <dl class="methodology">
            <dt v-text="t('team_activity.merged')" />
            <dd v-text="t('team_activity.pr_method')" />
            <dt v-text="t('team_activity.commits')" />
            <dd v-text="t('team_activity.commit_method')" />
            <dt v-text="t('team_activity.active_days')" />
            <dd v-text="t('team_activity.days_method')" />
            <dt v-text="t('team_activity.merge_time')" />
            <dd v-text="t('team_activity.merge_method')" />
          </dl>
          <details class="coverage-details">
            <summary v-text="t('team_activity.coverage')" />
            <ul>
              <li v-for="repo in report.repositories" :key="repo.id">
                <div>
                  <strong v-text="repo.full_name" /><span
                    class="coverage-status"
                    :data-status="repo.status"
                    v-text="repositoryStatus(repo.status)"
                  />
                </div>
                <p v-text="repo.default_branch || t('team_activity.default_branch')" />
                <p v-if="repo.issues.length" class="scope-issue" v-text="t('team_activity.scope_issue')" />
              </li>
            </ul>
          </details>
          <p class="data-note" v-text="t('team_activity.data_note')" />
        </section>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import Button from '~/components/atomic/Button.vue';
import Icon from '~/components/atomic/Icon.vue';
import type { IconNames } from '~/components/atomic/Icon.vue';
import TextField from '~/components/form/TextField.vue';
import useApiClient from '~/compositions/useApiClient';
import { useWPTitle } from '~/compositions/useWPTitle';
import type { TeamActivityReport, TeamActivityRepository } from '~/lib/api/types/teamActivity';
import { csvCell, rankMembers, summarizeActivity } from '~/lib/team-activity';

const { t, locale } = useI18n();
const apiClient = useApiClient();
const periods = [7, 30, 90];
const days = ref(30);
const repoId = ref('');
const timeZone = ref('Asia/Shanghai');
const timeZones = [
  ...new Set(['Asia/Shanghai', 'UTC', 'America/Los_Angeles', Intl.DateTimeFormat().resolvedOptions().timeZone]),
];
const excludeBots = ref(true);
const repositories = ref<TeamActivityRepository[]>([]);
const report = ref<TeamActivityReport>();
const loading = ref(false);
const errorMessage = ref('');
const memberSearch = ref('');
const rankBy = ref<'merged' | 'commits' | 'activeDays'>('merged');
const recentLimit = ref(8);
let generation = 0;
const summary = computed(() =>
  report.value ? summarizeActivity(report.value, timeZone.value, excludeBots.value) : undefined,
);
const rankedMembers = computed(() =>
  rankMembers(summary.value?.members ?? [], rankBy.value).filter((member) =>
    member.login.toLowerCase().includes(memberSearch.value.toLowerCase()),
  ),
);
const recentEvents = computed(() =>
  [...(summary.value?.events ?? [])]
    .sort((a, b) => Date.parse(b.occurred_at) - Date.parse(a.occurred_at))
    .slice(0, recentLimit.value),
);
const collectedRepositories = computed(
  () => report.value?.repositories.filter((repo) => repo.status === 'complete').length ?? 0,
);
const hasCoverage = computed(() => collectedRepositories.value > 0 || (summary.value?.events.length ?? 0) > 0);
const trendMax = computed(() => Math.max(1, ...(summary.value?.daily.map((day) => day.commits + day.merged) ?? [])));
const heatMax = computed(() => Math.max(1, ...(summary.value?.heatmap.flat() ?? [])));
const weekdays = computed(() =>
  Array.from({ length: 7 }, (_, day) =>
    new Intl.DateTimeFormat(locale.value, { weekday: 'short', timeZone: 'UTC' }).format(
      new Date(Date.UTC(2026, 10, 2 + day)),
    ),
  ),
);
const metrics = computed(() => {
  const data = summary.value;
  const value = (count?: number) => (hasCoverage.value ? number(count ?? 0) : '—');
  const list: { key: string; label: string; value: string; detail: string; icon: IconNames }[] = [
    {
      key: 'merged',
      label: t('team_activity.merged'),
      value: value(data?.merged),
      detail: t('team_activity.pr_authors'),
      icon: 'pull-request-closed',
    },
    {
      key: 'commits',
      label: t('team_activity.commits'),
      value: value(data?.commits),
      detail: t('team_activity.default_branch_only'),
      icon: 'commit',
    },
    {
      key: 'members',
      label: t('team_activity.active_members'),
      value: value(data?.members.length),
      detail: excludeBots.value ? t('team_activity.bots_excluded') : t('team_activity.bots_included'),
      icon: 'user',
    },
    {
      key: 'days',
      label: t('team_activity.active_days'),
      value: value(data?.activeDays),
      detail: t('team_activity.days_with_records'),
      icon: 'cron',
    },
    {
      key: 'merge_time',
      label: t('team_activity.merge_time'),
      value: hours(data?.medianMergeHours ?? null),
      detail: t('team_activity.median_detail'),
      icon: 'duration',
    },
    {
      key: 'repos',
      label: t('team_activity.active_repositories'),
      value: value(data?.activeRepositories),
      detail: t('team_activity.with_records'),
      icon: 'repo',
    },
  ];
  return list;
});
function number(value: number): string {
  return new Intl.NumberFormat(locale.value).format(value);
}
function hours(value: number | null): string {
  return value === null
    ? '—'
    : t('team_activity.hours', {
        value: new Intl.NumberFormat(locale.value, { maximumFractionDigits: 1 }).format(value),
      });
}
function formatDate(timestamp: string): string {
  return new Intl.DateTimeFormat(locale.value, {
    timeZone: timeZone.value,
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).format(new Date(timestamp));
}
function shortDate(timestamp: string): string {
  return new Intl.DateTimeFormat(locale.value, {
    timeZone: timeZone.value,
    month: 'short',
    day: 'numeric',
  }).format(new Date(timestamp));
}
function heatLevel(count: number): number {
  return count === 0 ? 0 : Math.max(1, Math.ceil((count / heatMax.value) * 4));
}
function avatarHue(login: string): number {
  return [...login].reduce((sum, char) => sum + char.charCodeAt(0), 0) % 360;
}
function safeURL(value: string): string | undefined {
  try {
    const url = new URL(value);
    return ['https:', 'http:'].includes(url.protocol) ? url.href : undefined;
  } catch {
    return undefined;
  }
}
function repositoryName(id: number): string {
  return report.value?.repositories.find((repo) => repo.id === id)?.full_name ?? String(id);
}
function repositoryStatus(status: TeamActivityRepository['status']): string {
  const labels = {
    complete: t('team_activity.complete'),
    partial: t('team_activity.incomplete'),
    skipped: t('team_activity.skipped'),
    pending: t('team_activity.pending'),
  };
  return labels[status];
}
async function load() {
  const request = ++generation;
  loading.value = true;
  errorMessage.value = '';
  recentLimit.value = 8;
  try {
    const result = await apiClient.getTeamActivity(days.value, repoId.value ? Number(repoId.value) : undefined);
    if (request === generation) {
      report.value = result;
      if (repoId.value === '') repositories.value = result.repositories;
    }
  } catch (error) {
    if (request === generation)
      errorMessage.value =
        error instanceof Error && error.message.includes('unsupported_forge')
          ? t('team_activity.unsupported')
          : t('team_activity.error');
  } finally {
    if (request === generation) loading.value = false;
  }
}
function exportCSV() {
  const headers = [
    t('team_activity.rank'),
    t('team_activity.member'),
    t('team_activity.merged'),
    t('team_activity.commits'),
    t('team_activity.active_days'),
    t('team_activity.repositories'),
    t('team_activity.merge_time'),
    t('team_activity.last_active'),
  ];
  const rows = rankedMembers.value.map((member) => [
    member.rank,
    member.login,
    member.merged,
    member.commits,
    member.activeDays,
    member.repositories,
    member.medianMergeHours ?? '',
    member.lastActive,
  ]);
  const csv = [headers, ...rows].map((row) => row.map(csvCell).join(',')).join('\r\n');
  const url = URL.createObjectURL(new Blob(['\uFEFF', csv], { type: 'text/csv;charset=utf-8;' }));
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `team-activity-${days.value}d.csv`;
  anchor.click();
  URL.revokeObjectURL(url);
}
watch([days, repoId], load);
onMounted(load);

onUnmounted(() => {
  generation++;
});
useWPTitle(computed(() => [t('team_activity.title')]));
</script>

<style scoped>
.team-page {
  --team-border: var(--wp-background-400);
  --team-muted: var(--wp-text-alt-100);
  --team-surface: var(--wp-background-100);
  --team-merge: #8059e6;
  --team-commit: var(--wp-accent);
  width: 100%;
  max-width: 1488px;
  margin: 0 auto;
  padding: 32px;
}
.team-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
.team-eyebrow {
  color: var(--wp-accent);
  font-size: 11px;
  font-weight: 650;
  margin-bottom: 6px;
}
h1 {
  color: var(--wp-text-200);
  font-size: 26px;
  font-weight: 650;
  line-height: 1.4;
}
.team-header p {
  color: var(--team-muted);
  font-size: 13px;
  margin-top: 6px;
}
.team-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  padding-bottom: 24px;
}
.period-switch {
  display: flex;
  padding: 3px;
  background: var(--wp-background-300);
  border-radius: 6px;
  gap: 2px;
}
.period-switch button {
  padding: 7px 13px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
}
.period-switch button[aria-pressed='true'] {
  background: var(--team-surface);
  color: var(--wp-accent);
  box-shadow: 0 1px 3px rgb(23 43 77 / 8%);
  font-weight: 600;
}
.toolbar-select {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 8px 10px;
  background: var(--team-surface);
  border: 1px solid var(--team-border);
  border-radius: 6px;
  font-size: 12px;
}
.toolbar-select :deep(svg) {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--team-muted);
}
.toolbar-select select {
  min-width: 0;
  max-width: 240px;
  background: transparent;
  cursor: pointer;
}
.bot-toggle {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-left: auto;
  font-size: 12px;
  cursor: pointer;
}
.bot-toggle input {
  width: 15px;
  height: 15px;
}
.scope-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 11px;
  color: var(--team-muted);
  margin-bottom: 14px;
}
.source-badge {
  padding: 3px 7px;
  border-radius: 4px;
  background: var(--wp-accent-soft);
  color: var(--wp-accent);
  font-weight: 600;
}
.scope-updated {
  margin-left: auto;
}
.coverage-warning {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  border: 1px solid var(--wp-hint-warn-200);
  border-radius: 6px;
  background: var(--wp-hint-warn-100);
  margin-bottom: 16px;
  font-size: 12px;
}
.coverage-warning :deep(svg) {
  flex-shrink: 0;
}
.coverage-warning p {
  margin-top: 4px;
}
.team-metrics {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}
.metric-card {
  display: flex;
  flex-direction: column;
  padding: 18px;
  background: var(--team-surface);
  border: 1px solid var(--team-border);
  border-radius: 7px;
  min-width: 0;
}
.metric-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  font-size: 12px;
  color: var(--team-muted);
}
.metric-label :deep(svg) {
  width: 17px;
  height: 17px;
}
.metric-card strong {
  font-size: 30px;
  line-height: 1.2;
  font-weight: 650;
  color: var(--wp-text-200);
  margin: 14px 0 8px;
  font-variant-numeric: tabular-nums;
}
.metric-footnote {
  font-size: 10px;
  color: var(--team-muted);
}
.metric-featured {
  background: var(--wp-accent-soft);
  border-color: var(--wp-kind-ci-border);
}
.metric-featured strong,
.metric-featured .metric-label {
  color: var(--wp-accent);
}
.team-main-grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}
.team-panel {
  min-width: 0;
  background: var(--team-surface);
  border: 1px solid var(--team-border);
  border-radius: 7px;
}
.panel-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 22px 16px;
}
.panel-heading h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--wp-text-200);
  font-size: 15px;
  font-weight: 600;
}
.panel-heading p {
  margin-top: 5px;
  font-size: 11px;
  color: var(--team-muted);
}
.panel-heading > :deep(svg) {
  color: var(--team-muted);
  width: 18px;
  height: 18px;
}
.chart-legend {
  display: flex;
  flex-shrink: 0;
  gap: 12px;
  font-size: 10px;
  color: var(--team-muted);
}
.chart-legend span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.chart-legend i {
  width: 7px;
  height: 7px;
  border-radius: 2px;
}
.legend-commit {
  background: var(--team-commit);
}
.legend-merge {
  background: var(--team-merge);
}
.daily-chart {
  display: flex;
  height: 160px;
  padding: 0 22px;
  gap: 12px;
}
.chart-scale {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  width: 20px;
  font-size: 10px;
  color: var(--team-muted);
}
.daily-columns {
  display: flex;
  flex: 1;
  align-items: flex-end;
  justify-content: space-around;
  gap: 4px;
  min-width: 0;
  border-bottom: 1px solid var(--team-border);
  background: repeating-linear-gradient(to top, transparent 0, transparent calc(50% - 1px), var(--team-border) 50%);
}
.daily-column {
  height: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  flex: 1;
  min-width: 0;
}
.daily-bar {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 20px;
  overflow: hidden;
  border-radius: 3px 3px 0 0;
}
.daily-bar > span {
  min-height: 0;
}
.bar-commit {
  background: var(--team-commit);
}
.bar-merge {
  background: var(--team-merge);
}
.chart-axis {
  display: flex;
  justify-content: space-between;
  padding: 8px 22px 18px 55px;
  color: var(--team-muted);
  font-size: 10px;
}
.heatmap-scroll {
  overflow-x: auto;
  padding: 0 22px;
}
.heatmap {
  min-width: 310px;
}
.heatmap-hours {
  display: grid;
  grid-template-columns: repeat(24, minmax(0, 1fr));
  margin-left: 34px;
  margin-bottom: 7px;
  color: var(--team-muted);
  font-size: 9px;
}
.heatmap-row {
  display: grid;
  grid-template-columns: 30px repeat(24, minmax(0, 1fr));
  gap: 3px;
  align-items: center;
  margin-bottom: 4px;
}
.weekday-label {
  color: var(--team-muted);
  font-size: 9px;
  line-height: 12px;
}
.heatmap-cell {
  display: block;
  height: 12px;
  border-radius: 2px;
  background: var(--wp-background-300);
}
.heatmap-cell[data-level='1'] {
  background: #d4e0ff;
}
.heatmap-cell[data-level='2'] {
  background: #9fb8ff;
}
.heatmap-cell[data-level='3'] {
  background: #648af8;
}
.heatmap-cell[data-level='4'] {
  background: #2d5cf6;
}
.heatmap-legend {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  padding: 10px 22px;
  color: var(--team-muted);
  font-size: 9px;
}
.heatmap-legend i {
  width: 10px;
  height: 10px;
}
.rhythm-note {
  padding: 0 22px 16px;
  color: var(--team-muted);
  font-size: 10px;
  line-height: 1.6;
}
.leaderboard-panel {
  margin-bottom: 20px;
}
.heading-count {
  font-size: 10px;
  color: var(--team-muted);
  background: var(--wp-background-200);
  padding: 2px 6px;
  border-radius: 4px;
}
.leaderboard-heading {
  align-items: center;
}
.ranking-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.member-search {
  max-width: 160px;
  font-size: 12px;
}
.rank-select {
  padding: 7px;
  max-width: 170px;
  border: 1px solid var(--team-border);
  border-radius: 5px;
  background: var(--team-surface);
  font-size: 12px;
}
.ranking-actions :deep(button) {
  font-size: 12px;
}
.leaderboard-scroll {
  overflow-x: auto;
}
.leaderboard-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  white-space: nowrap;
}
.leaderboard-table thead {
  background: var(--wp-background-200);
}
.leaderboard-table th {
  font-weight: 500;
}
.leaderboard-table thead th {
  padding: 11px 16px;
  color: var(--team-muted);
  text-align: right;
  font-size: 10px;
}
.leaderboard-table thead th:first-child,
.leaderboard-table thead th:nth-child(2) {
  text-align: left;
}
.leaderboard-table tbody td {
  padding: 13px 16px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  border-top: 1px solid var(--team-border);
}
.leaderboard-table tbody th {
  padding: 13px 16px;
  text-align: left;
  border-top: 1px solid var(--team-border);
}
.leaderboard-table tbody td:first-child {
  text-align: left;
  width: 60px;
}
.leaderboard-table tbody tr:hover {
  background: var(--wp-background-200);
}
.rank-number {
  display: inline-flex;
  justify-content: center;
  align-items: center;
  width: 24px;
  height: 24px;
  color: var(--team-muted);
  font-size: 11px;
}
.rank-top {
  background: var(--wp-accent-soft);
  color: var(--wp-accent);
  border-radius: 5px;
  font-weight: 650;
}
.member-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.member-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 7px;
  background: hsl(var(--avatar-hue) 55% 95%);
  color: hsl(var(--avatar-hue) 50% 35%);
  font-size: 10px;
  font-weight: 600;
}
.member-login {
  color: var(--wp-text-200);
  font-weight: 550;
}
.member-cell small {
  display: block;
  font-size: 9px;
  color: var(--team-muted);
  margin-top: 2px;
}
.sorted-value {
  color: var(--wp-accent);
  font-weight: 600;
}
.last-active {
  color: var(--team-muted);
  font-size: 10px;
}
.leaderboard-table .empty-table {
  text-align: center;
  padding: 36px;
  color: var(--team-muted);
}
.team-bottom-grid {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: 20px;
}
.recent-list {
  list-style: none;
  padding: 0 22px;
}
.recent-list li {
  display: flex;
  gap: 12px;
  border-top: 1px solid var(--team-border);
  padding: 13px 0;
}
.event-icon {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  border-radius: 6px;
  background: var(--wp-accent-soft);
  color: var(--wp-accent);
}
.event-icon :deep(svg) {
  width: 16px;
  height: 16px;
}
.event-icon.event-merged {
  background: var(--wp-kind-cd-background);
  color: var(--wp-kind-cd-text);
}
.event-content {
  min-width: 0;
  flex: 1;
}
.event-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 9px;
  color: var(--team-muted);
  margin-bottom: 4px;
}
.event-kind {
  font-weight: 500;
}
.event-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--wp-text-200);
  font-size: 12px;
}
.event-title:is(a):hover {
  color: var(--wp-accent);
  text-decoration: underline;
}
.event-content p {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  margin-top: 5px;
  gap: 8px;
  color: var(--team-muted);
  font-size: 10px;
  overflow-wrap: anywhere;
}
.event-separator {
  width: 1px;
  height: 8px;
  background: var(--team-border);
}
.more-events {
  width: 100%;
  padding: 12px;
  border-top: 1px solid var(--team-border);
  font-size: 12px;
  color: var(--wp-accent);
  cursor: pointer;
}
.methodology {
  padding: 0 22px;
}
.methodology dt {
  font-size: 11px;
  font-weight: 600;
  color: var(--wp-text-200);
  margin-bottom: 4px;
}
.methodology dd {
  font-size: 10px;
  color: var(--team-muted);
  line-height: 1.8;
  margin-bottom: 13px;
}
.coverage-details {
  padding: 0 22px;
  font-size: 11px;
}
.coverage-details summary {
  color: var(--wp-accent);
  cursor: pointer;
  margin-bottom: 10px;
}
.coverage-details ul {
  list-style: none;
}
.coverage-details li {
  border-top: 1px solid var(--team-border);
  padding: 10px 0;
}
.coverage-details li > div {
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: space-between;
}
.coverage-details strong {
  font-size: 10px;
  overflow-wrap: anywhere;
}
.coverage-details li p {
  color: var(--team-muted);
  font-size: 10px;
  margin-top: 4px;
}
.coverage-status {
  flex-shrink: 0;
  background: var(--wp-background-200);
  padding: 2px 5px;
  border-radius: 3px;
  font-size: 9px;
}
.coverage-status[data-status='complete'] {
  color: var(--wp-state-ok-100);
}
.scope-issue {
  color: var(--wp-error-100) !important;
}
.data-note {
  padding: 16px 22px;
  font-size: 10px;
  color: var(--team-muted);
  line-height: 1.8;
}
.team-state {
  min-height: 420px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  text-align: center;
  color: var(--team-muted);
}
.team-state > :deep(svg) {
  width: 32px;
  height: 32px;
  color: var(--wp-accent);
}
.team-state h2 {
  color: var(--wp-text-200);
  font-size: 17px;
  font-weight: 600;
}
.team-state p {
  max-width: 420px;
  font-size: 13px;
  line-height: 1.8;
}
.chart-empty {
  display: flex;
  min-height: 160px;
  align-items: center;
  justify-content: center;
  padding: 20px;
  color: var(--team-muted);
  font-size: 12px;
}
@media (max-width: 1200px) {
  .team-metrics {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .leaderboard-heading {
    align-items: flex-start;
    flex-wrap: wrap;
  }
  .team-main-grid {
    grid-template-columns: 1fr 1fr;
  }
}
@media (max-width: 800px) {
  .team-page {
    padding: 20px 16px;
  }
  .team-main-grid,
  .team-bottom-grid {
    grid-template-columns: 1fr;
  }
  .bot-toggle {
    margin-left: 0;
  }
  .ranking-actions {
    flex-wrap: wrap;
    width: 100%;
  }
  .member-search {
    flex: 1;
    max-width: none;
    min-width: 100px;
  }
  .team-header p {
    font-size: 11px;
  }
  .team-eyebrow {
    font-size: 10px;
  }
  h1 {
    font-size: 22px;
  }
  .metric-card {
    padding: 14px;
  }
  .metric-card strong {
    font-size: 25px;
  }
  .metric-label {
    font-size: 11px;
  }
  .metric-label :deep(svg) {
    display: none;
  }
  .scope-updated {
    width: 100%;
    margin-left: 0;
  }
}
@media (max-width: 430px) {
  .team-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
  .toolbar-select {
    flex: 1;
  }
  .toolbar-select select {
    width: 100%;
  }
  .team-toolbar {
    gap: 10px;
  }
  .panel-heading {
    padding: 18px 16px 14px;
  }
  .chart-legend {
    flex-direction: column;
    gap: 4px;
  }
  .period-switch {
    width: 100%;
  }
  .period-switch button {
    flex: 1;
  }
  .daily-chart {
    padding: 0 16px;
  }
  .daily-columns {
    gap: 2px;
  }
  .rank-select {
    max-width: 140px;
  }
  .heatmap-scroll {
    padding: 0 16px;
  }
  .metric-footnote {
    line-height: 1.5;
  }
}
</style>
