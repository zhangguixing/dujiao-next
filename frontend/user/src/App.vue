<template>
  <div id="app" class="min-h-screen bg-background text-foreground flex flex-col" :class="{ 'storefront-shell': !isResellerConsole }">
    <a v-if="!isResellerConsole" href="#storefront-main" class="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[70] focus:rounded-lg focus:bg-card focus:px-4 focus:py-3 focus:outline-2 focus:outline-primary">
      {{ t('common.skipToContent') }}
    </a>
    <Navbar v-if="!isResellerConsole" />
    <main id="storefront-main" tabindex="-1" class="flex-1 min-w-0">
      <ErrorBoundary>
        <RouterView v-slot="{ Component }">
          <Transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </ErrorBoundary>
    </main>
    <Footer v-if="!isResellerConsole" />
    <BackToTop v-if="!isResellerConsole" />
    <MobileBottomNav v-if="!isResellerConsole" />

    <Loading :loading="appStore.loading" />
    <Toast />
    <ConfirmDialog />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from './stores/app'
import Navbar from './components/Navbar.vue'
import Footer from './components/Footer.vue'
import Loading from './components/Loading.vue'
import Toast from './components/Toast.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import BackToTop from './components/BackToTop.vue'
import MobileBottomNav from './components/MobileBottomNav.vue'

// config 由 router.beforeEach 统一加载，无需在此重复调用
const appStore = useAppStore()
const { t } = useI18n()
const route = useRoute()
const isResellerConsole = computed(() => route.meta.resellerConsole === true)
</script>

<style>
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 200ms ease;
}

.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
}
</style>
