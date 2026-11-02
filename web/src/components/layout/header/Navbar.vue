<template>
  <nav
    class="navbar bg-wp-background-100 text-wp-text-100 dark:bg-wp-primary-300 border-wp-background-400 flex min-h-18 shrink-0 items-center gap-2 border-b px-4 py-3 md:px-6"
  >
    <div class="flex min-w-0 items-center gap-2 md:gap-5">
      <router-link
        :to="{ name: 'home' }"
        class="brand flex shrink-0 items-center gap-2.5 rounded-md"
        aria-label="Woodpecker CI"
      >
        <span class="bg-wp-control-info-100 flex h-10 w-10 items-center justify-center rounded-md">
          <WoodpeckerLogo class="h-7 w-7" aria-hidden="true" />
        </span>
        <span class="hidden flex-col sm:flex">
          <!-- eslint-disable-next-line @intlify/vue-i18n/no-raw-text -->
          <span class="text-wp-text-200 text-base leading-tight font-semibold">Woodpecker</span>
          <span class="text-wp-text-alt-100 text-xs leading-tight" :title="version?.current">{{
            version?.currentShort
          }}</span>
        </span>
      </router-link>
      <router-link
        v-if="user"
        :to="{ name: 'repos' }"
        class="navbar-clickable navbar-link"
        :class="{ 'navbar-link-active': route.path.startsWith(`${config.rootPath}/repos`) }"
      >
        <span class="flex md:hidden">{{ $t('repos') }}</span>
        <span class="hidden md:flex">{{ $t('repositories.title') }}</span>
      </router-link>
      <a
        href="https://woodpecker-ci.org/"
        target="_blank"
        rel="noopener noreferrer"
        class="navbar-clickable navbar-link hidden md:flex"
      >
        <span>{{ $t('docs') }}</span>
      </a>
      <a
        v-if="enableSwagger"
        :href="apiUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="navbar-clickable navbar-link hidden md:flex"
      >
        <span>{{ $t('api') }}</span>
      </a>
    </div>
    <div class="-m-1.5 ml-auto flex items-center space-x-2">
      <IconButton
        v-if="user?.admin"
        class="navbar-icon relative"
        :title="$t('settings')"
        :to="{ name: 'admin-settings' }"
      >
        <Icon name="settings" />
        <div v-if="version?.needsUpdate" class="bg-wp-error-100 absolute top-2 right-2 h-3 w-3 rounded-full" />
      </IconButton>

      <ActivePipelines v-if="user" class="navbar-icon p-1.5!" />
      <IconButton v-if="user" :to="{ name: 'user' }" :title="$t('user.settings.settings')" class="navbar-icon p-1.5!">
        <img v-if="user && user.avatar_url" class="rounded-md" :src="`${user.avatar_url}`" :alt="user.login" />
      </IconButton>
      <Button v-else :text="$t('login')" :to="{ name: 'login' }" color="blue" class="px-4 py-2" @click="saveRedirect" />
    </div>
  </nav>
</template>

<script lang="ts" setup>
import { useRoute } from 'vue-router';

import WoodpeckerLogo from '~/assets/logo.svg?component';
import Button from '~/components/atomic/Button.vue';
import Icon from '~/components/atomic/Icon.vue';
import IconButton from '~/components/atomic/IconButton.vue';
import useAuthentication from '~/compositions/useAuthentication';
import useConfig from '~/compositions/useConfig';
import useUserConfig from '~/compositions/useUserConfig';
import { useVersion } from '~/compositions/useVersion';

import ActivePipelines from './ActivePipelines.vue';

const version = useVersion();
const config = useConfig();
const userConfig = useUserConfig();
const route = useRoute();
const authentication = useAuthentication();
const { user } = authentication;
const apiUrl = `${config.rootPath ?? ''}/swagger/index.html`;

const { enableSwagger } = config;

function saveRedirect() {
  userConfig.setUserConfig('redirectUrl', route.fullPath);
}
</script>

<style scoped>
@reference '~/tailwind.css';

.navbar-icon {
  @apply hover:bg-wp-control-neutral-200! h-10 w-10 rounded-md p-2.5 dark:hover:bg-white/10!;
}

.navbar-icon :deep(svg) {
  @apply h-full w-full;
}

.navbar-link {
  @apply hover:bg-wp-control-neutral-200! rounded-md px-3 py-2 text-sm font-medium transition-colors dark:hover:bg-white/10!;
}
.navbar-link-active {
  color: var(--wp-accent);
  background: var(--wp-accent-soft);
}
</style>
