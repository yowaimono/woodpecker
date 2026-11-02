import { describe, expect, it } from 'vitest';

import type { Pipeline } from '~/lib/api/types';
import { WebhookEvents } from '~/lib/api/types';

import { classifyPipelineKind } from './pipeline-kind';

type Source = Pick<Pipeline, 'event' | 'ref' | 'deploy_to'>;
const source = (overrides: Partial<Source> = {}): Source => ({
  event: WebhookEvents.Push,
  ref: 'refs/heads/main',
  deploy_to: '',
  ...overrides,
});

describe('pipeline kind hints', () => {
  it.each([WebhookEvents.PullRequest, WebhookEvents.PullRequestClosed, WebhookEvents.PullRequestMetadata])(
    'marks %s as integration activity',
    (event) => {
      expect(classifyPipelineKind(source({ event }))).toEqual({ kind: 'ci', reason: 'pull_request' });
    },
  );

  it('marks an ordinary push as CI, including a branch named after production', () => {
    expect(classifyPipelineKind(source({ ref: 'refs/heads/prod-v2026.11.02.1' }))).toEqual({
      kind: 'ci',
      reason: 'push',
    });
  });

  it('gives an explicit deployment target precedence over a push event', () => {
    expect(classifyPipelineKind(source({ deploy_to: 'production' }))).toEqual({ kind: 'cd', reason: 'deployment' });
  });

  it('does not treat a whitespace-only target as deployment evidence', () => {
    expect(classifyPipelineKind(source({ deploy_to: '  ' })).kind).toBe('ci');
  });

  it.each([WebhookEvents.Deploy, WebhookEvents.Release])('recognizes the %s delivery event', (event) => {
    expect(classifyPipelineKind(source({ event })).kind).toBe('cd');
  });

  it.each([
    'refs/tags/prod-v2026.11.02.1',
    'refs/tags/prod-v2026.11.02.1-server',
    'refs/tags/prod-v2026.11.02.1-media-worker',
    'prod-v2026.11.02.1',
  ])('recognizes our production tag convention: %s', (ref) => {
    expect(classifyPipelineKind(source({ event: WebhookEvents.Tag, ref }))).toEqual({
      kind: 'cd',
      reason: 'production_tag',
    });
  });

  it('recognizes a manual rerun that retains a production tag ref', () => {
    expect(
      classifyPipelineKind(source({ event: WebhookEvents.Manual, ref: 'refs/tags/prod-v2026.11.02.1' })).kind,
    ).toBe('cd');
  });

  it.each(['refs/tags/v1.0.0', 'refs/tags/docs-update', 'refs/tags/prod-version', 'refs/tags/not-prod-v1'])(
    'does not assume every tag is CD: %s',
    (ref) => {
      expect(classifyPipelineKind(source({ event: WebhookEvents.Tag, ref }))).toEqual({
        kind: 'unknown',
        reason: 'unknown',
      });
    },
  );

  it.each([WebhookEvents.Cron, WebhookEvents.Manual])('does not guess the purpose of a %s run', (event) => {
    expect(classifyPipelineKind(source({ event })).kind).toBe('unknown');
  });

  it('tolerates missing optional runtime metadata in historical responses', () => {
    const historical = { event: WebhookEvents.Push } as Source;
    expect(classifyPipelineKind(historical).kind).toBe('ci');
  });
});
