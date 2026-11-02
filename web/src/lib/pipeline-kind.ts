import type { Pipeline } from '~/lib/api/types';

export type PipelineKind = 'ci' | 'cd' | 'unknown';
export type PipelineKindReason = 'deployment' | 'release' | 'production_tag' | 'pull_request' | 'push' | 'unknown';

type PipelineKindSource = Pick<Pipeline, 'event' | 'ref' | 'deploy_to'>;

export interface PipelineKindClassification {
  kind: PipelineKind;
  reason: PipelineKindReason;
}

/** UI hints follow our production-tag convention; event names alone cannot describe arbitrary workflow contents. */
export function classifyPipelineKind(pipeline: PipelineKindSource): PipelineKindClassification {
  if (pipeline.event === 'deployment' || pipeline.deploy_to?.trim()) {
    return { kind: 'cd', reason: 'deployment' };
  }

  if (pipeline.event === 'release') {
    return { kind: 'cd', reason: 'release' };
  }

  const ref = pipeline.ref?.trim() ?? '';
  const isTag = pipeline.event === 'tag' || ref.startsWith('refs/tags/');
  const tag = ref.replace(/^refs\/tags\//, '');
  if (isTag && /^prod-v\d[\w.+-]*$/i.test(tag)) {
    return { kind: 'cd', reason: 'production_tag' };
  }

  if (['pull_request', 'pull_request_closed', 'pull_request_metadata'].includes(pipeline.event)) {
    return { kind: 'ci', reason: 'pull_request' };
  }

  if (pipeline.event === 'push') {
    return { kind: 'ci', reason: 'push' };
  }

  return { kind: 'unknown', reason: 'unknown' };
}
