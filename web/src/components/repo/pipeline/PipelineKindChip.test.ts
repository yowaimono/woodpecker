import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { createI18n } from 'vue-i18n';

import en from '~/assets/locales/en.json';
import zh from '~/assets/locales/zh-Hans.json';
import { WebhookEvents } from '~/lib/api/types';

import PipelineKindChip from './PipelineKindChip.vue';

const mountChip = (locale = 'en') =>
  mount(PipelineKindChip, {
    props: { pipeline: { event: WebhookEvents.Push, ref: 'refs/heads/main', deploy_to: '' } },
    global: { plugins: [createI18n({ legacy: false, locale, messages: { en, 'zh-Hans': zh } })] },
  });

describe('pipeline kind chip', () => {
  it('shows a text label and explains the inference instead of relying on color alone', () => {
    const wrapper = mountChip();
    expect(wrapper.text()).toBe('CI');
    expect(wrapper.attributes('data-pipeline-kind')).toBe('ci');
    expect(wrapper.attributes('title')).toContain('push event');
    expect(wrapper.attributes('aria-label')).toBe(wrapper.attributes('title'));
    wrapper.unmount();
  });

  it('reacts when live pipeline metadata changes without remounting', async () => {
    const wrapper = mountChip();
    await wrapper.setProps({
      pipeline: { event: WebhookEvents.Deploy, ref: 'refs/heads/main', deploy_to: 'production' },
    });
    expect(wrapper.text()).toBe('CD');
    expect(wrapper.attributes('data-pipeline-kind')).toBe('cd');
    expect(wrapper.attributes('title')).toContain('Deployment');
    wrapper.unmount();
  });

  it('localizes the neutral fallback and its explanation', async () => {
    const wrapper = mountChip('zh-Hans');
    await wrapper.setProps({ pipeline: { event: WebhookEvents.Manual, ref: 'refs/heads/main', deploy_to: '' } });
    expect(wrapper.text()).toBe('未分类');
    expect(wrapper.attributes('data-pipeline-kind')).toBe('unknown');
    expect(wrapper.attributes('title')).toContain('信息不足');
    wrapper.unmount();
  });
});
