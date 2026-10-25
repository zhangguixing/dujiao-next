<template>
  <div :class="compact ? 'space-y-1' : 'space-y-3'">
    <a v-for="platform in linkedPlatforms" :key="platform" :href="contact[platform]" target="_blank" rel="noopener noreferrer" :class="entryClass">
      <SocialIcon :platform="platform" />
      <span>{{ t(`contactChannels.${platform}`) }}</span>
    </a>
    <Popover v-for="platform in accountPlatforms" :key="platform" @update:open="resetFeedback">
      <PopoverTrigger as-child>
        <button type="button" :class="entryClass" :aria-label="t('contactChannels.showAccount', { platform: t(`contactChannels.${platform}`) })">
          <SocialIcon :platform="platform" />
          <span>{{ t(`contactChannels.${platform}`) }}</span>
        </button>
      </PopoverTrigger>
      <PopoverContent side="top" align="start" :collision-padding="12" class="max-h-[var(--reka-popover-content-available-height)] max-w-[calc(100vw-2rem)] space-y-4 overflow-y-auto" :aria-label="t(`contactChannels.${platform}`)">
        <div class="flex items-center gap-2 font-semibold"><SocialIcon :platform="platform" /><span>{{ t(`contactChannels.${platform}`) }}</span></div>
        <div v-if="contact[platform]" class="space-y-2">
          <p class="text-xs text-muted-foreground">{{ t(`contactChannels.${platform}Account`) }}</p>
          <p class="break-all select-text text-base font-medium">{{ contact[platform] }}</p>
          <Button type="button" variant="outline" size="sm" class="min-h-11 w-full" :disabled="copying" @click="copyAccount(platform)"><Copy class="h-4 w-4" />{{ t('contactChannels.copyAccount') }}</Button>
        </div>
        <div v-if="platform === 'wechat' && contact.wechat_qrcode" class="space-y-2">
          <img v-if="!qrImageFailed" :src="getImageUrl(contact.wechat_qrcode)" :alt="t('contactChannels.wechatQRCode')" class="mx-auto max-h-64 w-full rounded-md bg-white object-contain p-2" @error="qrImageFailed = true" />
          <p v-if="qrImageFailed" class="text-sm text-destructive" role="alert">{{ t('contactChannels.qrLoadFailed') }}</p>
          <p v-else class="text-xs text-muted-foreground">{{ t('contactChannels.scanWeChat') }}</p>
        </div>
        <p v-if="feedback" class="text-xs" :class="copyFailed ? 'text-destructive' : 'text-muted-foreground'" role="status" aria-live="polite">{{ feedback }}</p>
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Copy } from 'lucide-vue-next'
import { Button } from './ui/button'
import { Popover, PopoverContent, PopoverTrigger } from './ui/popover'
import SocialIcon from './SocialIcon.vue'
import { useAppStore } from '../stores/app'
import { copyText } from '../utils/clipboard'
import { getImageUrl } from '../utils/image'

const props = withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
const { t } = useI18n()
const appStore = useAppStore()
const contact = computed<Record<string, string>>(() => appStore.config?.contact || {})
const linkedPlatforms = computed(() => (['telegram', 'whatsapp'] as const).filter(platform => contact.value[platform]))
const accountPlatforms = computed(() => (['qq', 'wechat'] as const).filter(platform => contact.value[platform] || (platform === 'wechat' && contact.value.wechat_qrcode)))
const entryClass = computed(() => props.compact
  ? 'flex min-h-11 w-full items-center gap-2 py-2 text-left text-sm text-muted-foreground transition-colors hover:text-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary'
  : 'flex min-h-11 w-full items-center gap-3 rounded-lg border border-border bg-secondary/40 p-3 text-left text-sm text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary')
const copying = ref(false)
const feedback = ref('')
const copyFailed = ref(false)
const qrImageFailed = ref(false)
watch(() => contact.value.wechat_qrcode, () => { qrImageFailed.value = false })
const resetFeedback = () => { feedback.value = ''; copyFailed.value = false }
const copyAccount = async (platform: 'qq' | 'wechat') => {
  if (copying.value || !contact.value[platform]) return
  copying.value = true
  resetFeedback()
  try {
    await copyText(contact.value[platform])
    feedback.value = t('contactChannels.copied')
  } catch {
    copyFailed.value = true
    feedback.value = t('contactChannels.copyFailed')
  } finally {
    copying.value = false
  }
}
</script>
