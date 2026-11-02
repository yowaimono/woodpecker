<template>
  <main class="login-page flex min-h-full w-full flex-col items-center justify-center gap-4 px-4 py-8">
    <Error v-if="errorMessage" class="w-full max-w-3xl">
      <div class="flex min-w-0 flex-col gap-1 text-left">
        <span class="break-words whitespace-pre-wrap">{{ errorMessage }}</span>
        <span v-if="errorDescription" class="break-words whitespace-pre-wrap">{{ errorDescription }}</span>
        <a
          v-if="errorUri"
          :href="errorUri"
          target="_blank"
          rel="noopener noreferrer"
          class="cursor-pointer break-all text-white underline hover:opacity-80"
        >
          <span>{{ errorUri }}</span>
        </a>
      </div>
    </Error>

    <div
      class="login-card border-wp-background-400 bg-wp-background-100 dark:bg-wp-background-200 flex w-full max-w-3xl flex-col overflow-hidden rounded-lg border md:min-h-96 md:flex-row"
    >
      <div class="login-brand flex min-h-56 flex-col items-center justify-center gap-6 px-8 py-10 md:w-1/2">
        <div class="bg-wp-control-info-100 flex h-24 w-24 items-center justify-center rounded-lg">
          <WoodpeckerLogo preserveAspectRatio="xMinYMin slice" class="h-16 w-16" aria-hidden="true" />
        </div>
        <!-- eslint-disable-next-line @intlify/vue-i18n/no-raw-text -->
        <span class="text-wp-link-100 text-2xl font-semibold">Woodpecker CI</span>
        <div class="login-pipeline flex items-center" aria-hidden="true">
          <span class="login-node"><Icon name="repo" /></span>
          <span class="login-connector" />
          <span class="login-node"><Icon name="status-success" /></span>
          <span class="login-connector" />
          <span class="login-node"><Icon name="deployment" /></span>
        </div>
      </div>
      <div class="flex min-h-64 flex-col items-center justify-center gap-7 px-6 py-10 text-center md:w-1/2 md:px-8">
        <h1 class="text-wp-text-200 text-xl leading-relaxed font-semibold">{{ $t('login_to_woodpecker_with') }}</h1>
        <div class="flex w-full max-w-xs flex-col gap-3">
          <Button
            v-for="forge in forgesWithNameAndFavicon"
            :key="forge.id"
            :start-icon="forge.type === 'addon' ? 'repo' : forge.type"
            color="blue"
            class="min-h-11 justify-center gap-2 px-4 py-2.5 whitespace-normal!"
            @click="authenticate(forge.id)"
          >
            <div class="mr-2 w-4">
              <img
                v-if="forge.favicon && !failedForgeFavicons.has(forge.id)"
                :src="forge.favicon"
                :alt="$t('login_to_woodpecker_with', { forge: forge.name })"
                @error="() => failedForgeFavicons.add(forge.id)"
              />
              <Icon v-else :name="forge.type === 'addon' ? 'repo' : forge.type" />
            </div>

            {{ forge.name }}
          </Button>
        </div>
      </div>
    </div>
  </main>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import WoodpeckerLogo from '~/assets/logo.svg?component';
import Button from '~/components/atomic/Button.vue';
import Error from '~/components/atomic/Error.vue';
import Icon from '~/components/atomic/Icon.vue';
import useApiClient from '~/compositions/useApiClient';
import useAuthentication from '~/compositions/useAuthentication';
import { useWPTitle } from '~/compositions/useWPTitle';
import { forgeDisplayName } from '~/lib/forgeDisplayName';
import type { Forge } from '~/lib/api/types';

const route = useRoute();
const { authenticate } = useAuthentication();
const i18n = useI18n();
const apiClient = useApiClient();

const forges = ref<Forge[]>([]);

const authErrorMessages = {
  oauth_error: i18n.t('oauth_error'),
  internal_error: i18n.t('internal_error'),
  registration_closed: i18n.t('registration_closed'),
  access_denied: i18n.t('access_denied'),
  invalid_state: i18n.t('invalid_state'),
  org_access_denied: i18n.t('org_access_denied'),
};

const errorMessage = ref<string>();
const errorDescription = ref<string>(route.query.error_description as string);
// only link http(s) URIs, anything else (e.g. javascript:) could run code on click
const errorUri = computed(() => {
  const uri = route.query.error_uri as string | undefined;
  return uri?.startsWith('http') ? uri : undefined;
});

onMounted(async () => {
  forges.value = (await apiClient.getForges()) ?? [];

  if (route.query.error) {
    const error = route.query.error as keyof typeof authErrorMessages;
    errorMessage.value = authErrorMessages[error] ?? error;
  }
});

useWPTitle(computed(() => [i18n.t('login')]));

const failedForgeFavicons = ref(new Set<number>()); // Track which favicons failed to load

const forgesWithNameAndFavicon = computed(() =>
  forges.value.map((forge) => {
    let favicon: null | string = null;

    if ((forge.url !== undefined && forge.url !== '') || (forge.oauth_host !== undefined && forge.oauth_host !== '')) {
      const url = new URL(forge.oauth_host ?? forge.url);
      favicon = `${url.origin}/favicon.ico`;
    }

    return {
      ...forge,
      name: forgeDisplayName(forge),
      favicon,
    };
  }),
);
</script>

<style scoped>
.login-card {
  box-shadow: 0 12px 40px rgb(23 43 77 / 5%);
}

.login-brand {
  background: var(--wp-accent-soft);
}

.login-node {
  display: flex;
  width: 2.25rem;
  height: 2.25rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--wp-control-neutral-300);
  border-radius: 6px;
  background: var(--wp-background-100);
  color: var(--wp-accent);
}

.login-connector {
  width: 2rem;
  height: 1px;
  background: var(--wp-control-neutral-300);
}
</style>
