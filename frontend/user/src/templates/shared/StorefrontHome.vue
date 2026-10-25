<template>
  <div class="custom-storefront" :class="`custom-storefront--${variant}`">
    <section class="storefront-intro" :aria-labelledby="`${variant}-title`">
      <div class="storefront-intro-copy">
        <p class="storefront-eyebrow"><span></span>{{ t(`storefrontThemes.${variant}.eyebrow`) }}</p>
        <h1 :id="`${variant}-title`">{{ t(`storefrontThemes.${variant}.title`) }}<em>{{ t(`storefrontThemes.${variant}.highlight`) }}</em></h1>
        <p class="storefront-description">{{ description || t(`storefrontThemes.${variant}.description`) }}</p>
        <div class="storefront-actions">
          <RouterLink :to="catalogTarget" class="storefront-primary">{{ t('home.hero.cta') }}<ArrowUpRight :size="18" /></RouterLink>
          <RouterLink to="/guest/orders" class="storefront-secondary">{{ t('navbar.guestOrders') }}<ArrowRight :size="16" /></RouterLink>
        </div>
      </div>
      <div v-if="variant === 'mint'" class="mint-object" aria-hidden="true">
        <div class="mint-orbit"></div>
        <div class="mint-ticket mint-ticket-back"><Layers :size="44" /></div>
        <div class="mint-ticket mint-ticket-front"><Package :size="66" :stroke-width="1" /><span>{{ t('storefrontThemes.mint.objectLabel') }}</span><ArrowUpRight :size="24" /></div>
        <span class="mint-spark">✳</span>
      </div>
      <div v-else-if="variant === 'paper'" class="paper-index">
        <span class="paper-index-label">{{ t('storefrontThemes.paper.indexLabel') }}</span>
        <RouterLink :to="catalogTarget"><span>01</span>{{ t('nav.products') }}<ArrowUpRight :size="20" /></RouterLink>
        <RouterLink to="/cart"><span>02</span>{{ t('navbar.cart') }}<ArrowUpRight :size="20" /></RouterLink>
        <RouterLink to="/guest/orders"><span>03</span>{{ t('navbar.guestOrders') }}<ArrowUpRight :size="20" /></RouterLink>
        <div class="paper-stamp" aria-hidden="true"><Asterisk :size="74" :stroke-width="1" /></div>
      </div>
      <div v-else class="mono-object" aria-hidden="true">
        <div class="mono-cross mono-cross-a">+</div><div class="mono-cross mono-cross-b">+</div>
        <div class="mono-cube"><div></div><div></div><div></div></div>
        <span class="mono-caption">{{ t('storefrontThemes.mono.objectLabel') }}<ArrowDownRight :size="16" /></span>
      </div>
    </section>
    <div class="storefront-divider"><span>{{ t(`storefrontThemes.${variant}.collection`) }}</span><span>{{ t('home.featured.description') }}</span></div>
    <div id="storefront-catalog"><VaultHome /></div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDownRight, ArrowRight, ArrowUpRight, Asterisk, Layers, Package } from 'lucide-vue-next'
import { useAppStore } from '../../stores/app'
import { getLocalizedText } from '../../utils/resellerSiteConfig'
import VaultHome from '../vault/Home.vue'

defineProps<{ variant: 'mint' | 'paper' | 'mono' }>()
const { t } = useI18n()
const appStore = useAppStore()
const description = computed(() => getLocalizedText(appStore.config?.brand?.site_description, appStore.locale))
const catalogTarget = computed(() => appStore.config?.template_mode === 'list' ? '/#storefront-catalog' : '/products')
</script>
