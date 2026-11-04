<template>
  <span
    class="pipeline-kind-chip"
    role="note"
    :data-pipeline-kind="classification.kind"
    :title="description"
    :aria-label="description"
  >
    {{ label }}
  </span>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { Pipeline } from '~/lib/api/types';
import { classifyPipelineKind } from '~/lib/pipeline-kind';

const props = defineProps<{
  pipeline: Pick<Pipeline, 'event' | 'ref' | 'deploy_to'>;
}>();

const { t } = useI18n();
const classification = computed(() => classifyPipelineKind(props.pipeline));
const kindInfo = computed(() => {
  const kinds = {
    ci: { label: t('repo.pipeline.kind.ci'), description: t('repo.pipeline.kind.ci_description') },
    cd: { label: t('repo.pipeline.kind.cd'), description: t('repo.pipeline.kind.cd_description') },
    unknown: { label: t('repo.pipeline.kind.unknown'), description: t('repo.pipeline.kind.unknown_description') },
  };
  return kinds[classification.value.kind];
});
const label = computed(() => kindInfo.value.label);
const description = computed(() => {
  const reasons = {
    deployment: t('repo.pipeline.kind.reasons.deployment'),
    release: t('repo.pipeline.kind.reasons.release'),
    production_tag: t('repo.pipeline.kind.reasons.production_tag'),
    pull_request: t('repo.pipeline.kind.reasons.pull_request'),
    push: t('repo.pipeline.kind.reasons.push'),
    unknown: t('repo.pipeline.kind.reasons.unknown'),
  };
  return t('repo.pipeline.kind.description', {
    kind: kindInfo.value.description,
    reason: reasons[classification.value.reason],
  });
});
</script>

<style scoped>
.pipeline-kind-chip {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  width: fit-content;
  padding: 0.125rem 0.5rem;
  border: 1px solid var(--wp-kind-unknown-border);
  border-radius: 9999px;
  background: var(--wp-kind-unknown-background);
  color: var(--wp-kind-unknown-text);
  font-size: 0.6875rem;
  font-weight: 600;
  line-height: 1rem;
  white-space: nowrap;
}

.pipeline-kind-chip[data-pipeline-kind='ci'] {
  border-color: var(--wp-kind-ci-border);
  background: var(--wp-kind-ci-background);
  color: var(--wp-kind-ci-text);
}

.pipeline-kind-chip[data-pipeline-kind='cd'] {
  border-color: var(--wp-kind-cd-border);
  background: var(--wp-kind-cd-background);
  color: var(--wp-kind-cd-text);
}
</style>
