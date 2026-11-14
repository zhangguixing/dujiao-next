<template>
  <div class="home-page min-h-screen bg-background text-foreground">

    <section v-if="displayHero" class="home-opening storefront-container pt-24 sm:pt-28">
      <div
        class="home-hero"
        :class="{ 'home-hero--image': heroImage }"
        @touchstart.passive="onBannerTouchStart"
        @touchend.passive="onBannerTouchEnd"
      >
        <div class="home-hero-light" aria-hidden="true">
          <span class="home-orbit"></span>
        </div>
        <Transition name="banner-fade">
          <img v-if="heroImage" :key="heroImage" :src="heroImage" alt="" class="home-banner-image" />
        </Transition>
        <div v-if="heroImage" class="home-hero-scrim" aria-hidden="true"></div>

        <div v-if="bannerLoading" class="home-hero-content" :aria-label="t('common.loading')" aria-busy="true">
          <div class="home-hero-skeleton h-12 w-2/3 max-w-md rounded-lg"></div>
          <div class="home-hero-skeleton mt-5 h-5 w-1/2 max-w-sm rounded-lg"></div>
          <div class="home-hero-skeleton mt-8 h-11 w-32 rounded-lg"></div>
        </div>
        <div v-else class="home-hero-content">
          <h1 class="home-hero-title">{{ displayHeroTitle }}</h1>
          <p class="home-hero-subtitle">{{ heroSubtitle }}</p>
          <div class="mt-7 flex flex-wrap items-center gap-3">
            <a v-if="!hasHeroLink && templateMode === 'list'" href="#home-catalog" class="home-hero-primary">
              {{ heroPrimaryButtonText }}
              <ArrowRight class="size-4" aria-hidden="true" />
            </a>
            <router-link v-else-if="!hasHeroLink" to="/products" class="home-hero-primary">
              {{ heroPrimaryButtonText }}
              <ArrowRight class="size-4" aria-hidden="true" />
            </router-link>
            <button v-else type="button" class="home-hero-primary" @click="goToHeroLink">
              {{ heroPrimaryButtonText }}
              <ArrowRight class="size-4" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div v-if="bannerCount > 1" class="home-banner-controls">
          <button type="button" :aria-label="t('common.previousBanner')" @click="handlePrevHeroBanner">
            <ChevronLeft class="size-4" aria-hidden="true" />
          </button>
          <button
            v-for="(_, index) in banners"
            :key="index"
            type="button"
            class="home-banner-step"
            :aria-current="index === currentBannerIndex ? 'true' : undefined"
            :aria-label="t('common.switchBanner', { n: index + 1 })"
            @click="selectHeroBanner(index)"
          ></button>
          <button type="button" :aria-label="t('common.nextBanner')" @click="handleNextHeroBanner">
            <ChevronRight class="size-4" aria-hidden="true" />
          </button>
        </div>
      </div>
    </section>

    <!-- ==================== LIST MODE ==================== -->
    <template v-if="templateMode === 'list'">
      <!-- Main: Left Categories + Right Product List -->
      <section id="home-catalog" class="home-catalog relative z-10 pb-6" :class="displayHero ? 'pt-6' : 'pt-24'">
        <div class="storefront-container">
          <div class="flex flex-col lg:flex-row gap-6">

            <CategorySidebar
              :categories="listCategoryGroups"
              :selected-category="listSelectedCategory"
              :expanded-parent-ids="listExpandedParentIds"
              :show-drawer="listShowFilterDrawer"
              compact
              @select-category="listSelectCategory"
              @toggle-parent="listToggleParentCategory"
              @update:show-drawer="listShowFilterDrawer = $event"
            />

            <!-- Right: Product List -->
            <div class="flex-1 min-w-0">
              <!-- Search Bar -->
              <div class="relative mb-4">
                <div class="absolute inset-y-0 left-3.5 flex items-center pointer-events-none">
                  <Search class="w-4 h-4 text-muted-foreground" />
                </div>
                <input
                  v-model="listSearchQuery"
                  type="text"
                  class="w-full h-11 pl-10 pr-10 rounded-xl border bg-card text-sm focus:outline-none focus:ring-2 focus:ring-primary/30 text-foreground placeholder:text-muted-foreground transition-shadow"
                  :aria-label="t('products.searchLabel')"
                  :placeholder="t('products.searchBoxPlaceholder')"
                  @keydown.enter="listOnSearch"
                />
                <button
                  v-if="listSearchQuery"
                  type="button"
                  class="absolute inset-y-0 right-3 flex items-center text-muted-foreground hover:text-foreground transition-colors"
                  :aria-label="t('products.clearSearch')"
                  @click="listClearSearch"
                >
                  <X class="w-4 h-4" />
                </button>
              </div>

              <!-- Loading Skeleton -->
              <div v-if="listLoading" class="space-y-6">
                <div v-for="i in 3" :key="i">
                  <div class="flex items-center gap-2 mb-3 px-0.5">
                    <div class="h-5 w-5 rounded theme-skeleton"></div>
                    <div class="h-4 w-28 rounded theme-skeleton"></div>
                  </div>
                  <div class="space-y-2">
                    <div v-for="j in 3" :key="j"
                      class="bg-card rounded-xl border flex items-center h-[72px]">
                      <div class="w-14 h-14 m-2 rounded-lg theme-skeleton flex-shrink-0"></div>
                      <div class="flex-1 px-3 py-2 space-y-2">
                        <div class="h-3.5 w-1/3 rounded theme-skeleton"></div>
                        <div class="h-3 w-1/4 rounded theme-skeleton"></div>
                      </div>
                      <div class="px-4 py-2">
                        <div class="h-4 w-14 rounded theme-skeleton"></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Grouped Product List -->
              <div v-else-if="listProductGroups.length > 0" class="space-y-6">
                <div v-for="group in listProductGroups" :key="group.categoryId ?? 'uncategorized'">
                  <!-- Category Header -->
                  <div class="flex items-center gap-2 mb-3 px-0.5">
                    <span class="w-1 h-5 rounded-full bg-primary flex-shrink-0"></span>
                    <img v-if="group.categoryIcon" :src="getImageUrl(group.categoryIcon)"
                      :alt="group.categoryName" loading="lazy" class="h-5 w-5 rounded object-cover flex-shrink-0" />
                    <span class="text-sm font-semibold text-foreground truncate">{{ group.categoryName }}</span>
                    <span class="text-xs text-muted-foreground">({{ group.products.length }})</span>
                  </div>
                  <!-- Products in this category -->
                  <div class="space-y-2">
                    <ProductListItem
                      v-for="(product, idx) in group.products"
                      :key="product.id"
                      :product="product"
                      :index="idx"
                      :animation-step="20"
                      @click="goToProduct"
                      @quick-buy="openQuickBuy"
                    />
                  </div>
                </div>

                <PaginationNav
                  :current-page="listCurrentPage"
                  :total-pages="listTotalPages"
                  :loading="listLoading"
                  compact
                  @change-page="listChangePage"
                />
              </div>

              <!-- Empty State -->
              <EmptyState v-else variant="soft" icon="package" :title="t('products.empty')" />
            </div>
          </div>
        </div>
      </section>
    </template>

    <!-- ==================== CARD MODE (default) ==================== -->
    <template v-else>
    <section id="featured" class="home-catalog relative z-10 pb-14" :class="displayHero ? 'pt-8 sm:pt-12' : 'pt-24 sm:pt-28'">
      <div class="storefront-container">
        <div class="home-catalog-heading storefront-page-heading flex flex-wrap items-end justify-between gap-3">
          <div>
            <component :is="displayHero ? 'h2' : 'h1'" class="storefront-page-title">{{ t('home.featured.title') }}</component>
            <p class="storefront-page-subtitle">{{ t('home.featured.description') }}</p>
          </div>
          <router-link
            to="/products"
            class="home-catalog-link inline-flex min-h-11 items-center gap-2 rounded-lg px-3 text-sm font-semibold text-primary focus-visible:outline-2 focus-visible:outline-primary"
          >
            {{ t('home.featured.viewAll') }}
            <ArrowRight class="size-4" aria-hidden="true" />
          </router-link>
        </div>

        <div v-if="featuredLoading" class="grid grid-cols-2 gap-3 md:grid-cols-3 md:gap-4 lg:grid-cols-4" :aria-label="t('common.loading')" aria-busy="true">
          <div v-for="index in 4" :key="index" class="storefront-surface overflow-hidden">
            <div class="theme-skeleton aspect-[4/3]"></div>
            <div class="space-y-3 p-4">
              <div class="theme-skeleton h-4 w-3/4"></div>
              <div class="theme-skeleton h-5 w-1/2"></div>
            </div>
          </div>
        </div>
        <EmptyState v-else-if="featuredError" icon="alert" :title="t('common.api.requestFailed')">
          <template #action>
            <Button variant="secondary" @click="loadFeaturedProducts">{{ t('errorBoundary.retry') }}</Button>
          </template>
        </EmptyState>
        <div v-else-if="products.length > 0" class="grid grid-cols-2 gap-3 md:grid-cols-3 md:gap-4 lg:grid-cols-4">
          <ProductCard
            v-for="(product, idx) in products"
            :key="product.id"
            :product="product"
            :index="idx"
            :animation-step="60"
            class="home-product"
            @click="goToProduct"
            @quick-buy="openQuickBuy"
          />
        </div>
        <div v-else class="home-empty rounded-2xl border py-16 text-center text-muted-foreground theme-slide-up">
          <PackageOpen class="mx-auto h-16 w-16 mb-4 text-muted-foreground opacity-50" :stroke-width="1.5" />
          {{ t('home.featured.empty') }}
        </div>
      </div>
    </section>

    <template v-if="latestSectionVisible">
    <hr class="theme-section-divider mx-4 md:mx-auto md:max-w-6xl" />

    <section class="relative z-10 py-12">
      <div class="storefront-container">
        <div class="mb-6 flex items-end justify-between gap-4">
          <div>
            <h2 class="theme-section-heading text-[1.7rem]">{{ t('home.latest.title') }}</h2>
            <p class="mt-1 text-sm text-muted-foreground">{{ t('home.latest.description') }}</p>
          </div>
          <div class="flex items-center gap-3 text-sm">
            <router-link v-if="blogEnabled" to="/blog" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.blog') }}</router-link>
            <router-link v-if="noticeEnabled" to="/notice" class="text-muted-foreground transition-colors hover:text-foreground">{{ t('nav.notice') }}</router-link>
          </div>
        </div>

        <div v-if="posts.length > 0" class="grid grid-cols-1 gap-5 md:grid-cols-3">
          <article
            v-for="post in posts"
            :key="post.id"
            class="home-post rounded-xl border bg-card p-5"
          >
            <div class="mb-2 text-xs text-muted-foreground">{{ formatDate(post.published_at) }}</div>
            <h3 class="line-clamp-2 text-base font-semibold">{{ getLocalizedText(post.title) }}</h3>
            <p class="mt-2 line-clamp-2 text-sm text-muted-foreground">{{ getLocalizedText(post.summary) }}</p>
            <router-link :to="`/blog/${post.slug}`" class="mt-4 inline-flex items-center gap-2 text-sm font-semibold text-primary after:absolute after:inset-0 hover:underline focus-visible:outline-2 focus-visible:outline-primary">
              {{ t('blog.readMore') }}
              <ArrowRight class="size-4" aria-hidden="true" />
            </router-link>
          </article>
        </div>
        <div v-else class="rounded-2xl border border-dashed py-12 text-center text-muted-foreground">
          {{ t('blog.empty') }}
        </div>
      </div>
    </section>
    </template>
    </template>

    <ProductQuickBuy
      v-if="quickBuyProduct"
      :product="quickBuyProduct"
      :visible="quickBuyVisible"
      @update:visible="quickBuyVisible = $event"
    />

    <AnnouncementModal
      v-if="activeAnnouncement"
      :announcement="activeAnnouncement"
      :visible="announcementVisible"
      @update:visible="announcementVisible = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowRight, ChevronLeft, ChevronRight, PackageOpen, Search, X } from 'lucide-vue-next'
import { postAPI, productAPI } from '../api'
import { getImageUrl } from '../utils/image'
import { useLocalized } from '../composables/useProduct'
import { useBannerCarousel } from '../composables/useBannerCarousel'
import { useProductList } from '../composables/useProductList'
import { useProductListGroups } from '../composables/useProductListGroups'
import { usePageSeo } from '../composables/usePageSeo'
import { useAppStore } from '../stores/app'
import ProductCard from '../components/ProductCard.vue'
import ProductListItem from '../components/ProductListItem.vue'
import ProductQuickBuy from '../components/ProductQuickBuy.vue'
import CategorySidebar from '../components/CategorySidebar.vue'
import PaginationNav from '../components/PaginationNav.vue'
import EmptyState from '../components/EmptyState.vue'
import { Button } from '@/components/ui/button'
import AnnouncementModal from '../components/AnnouncementModal.vue'
import { recordHomeVisit } from '../utils/homeVisit'
import { useAnnouncement, type HomeAnnouncement } from '../composables/useAnnouncement'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const { getLocalizedText } = useLocalized()
const appStore = useAppStore()

const templateMode = computed(() => appStore.config?.template_mode || 'card')
const navBuiltin = computed(() => (appStore.config?.nav_config as { builtin?: Record<string, boolean> } | undefined)?.builtin)
const blogEnabled = computed(() => navBuiltin.value?.blog !== false)
const noticeEnabled = computed(() => navBuiltin.value?.notice !== false)
const latestSectionVisible = computed(() => blogEnabled.value || noticeEnabled.value)

// ==================== Shared State ====================
const products = ref<any[]>([])
const featuredLoading = ref(true)
const featuredError = ref(false)
const posts = ref<any[]>([])
const quickBuyProduct = ref<any>(null)
const quickBuyVisible = ref(false)

const { shouldShow } = useAnnouncement()
const activeAnnouncement = ref<HomeAnnouncement | null>(null)
const announcementVisible = ref(false)

const openQuickBuy = (product: any) => {
  quickBuyProduct.value = product
  quickBuyVisible.value = true
}

// ==================== Banner Carousel ====================
const {
  banners,
  bannerLoading,
  currentBannerIndex,
  bannerCount,
  showHeroSection,
  heroImage,
  heroTitle,
  heroSubtitle,
  hasHeroLink,
  heroPrimaryButtonText,
  loadBanners,
  handleNextHeroBanner,
  handlePrevHeroBanner,
  selectHeroBanner,
  goToHeroLink,
  onBannerTouchStart,
  onBannerTouchEnd,
  stopHeroAutoPlay,
} = useBannerCarousel()

const isHome = computed(() => route.name === 'home')
const displayHero = computed(() => isHome.value || showHeroSection.value)
const displayHeroTitle = computed(() => {
  if (bannerCount.value) return heroTitle.value
  return String(appStore.config?.brand?.site_name || '').trim() || heroTitle.value
})

// ==================== List Mode ====================
const {
  loading: listLoading,
  products: listProducts,
  selectedCategory: listSelectedCategory,
  searchQuery: listSearchQuery,
  currentPage: listCurrentPage,
  totalPages: listTotalPages,
  showFilterDrawer: listShowFilterDrawer,
  expandedParentIds: listExpandedParentIds,
  categoryGroups: listCategoryGroups,
  categoryMap: listCategoryMap,
  selectCategory: listSelectCategory,
  toggleParentCategory: listToggleParentCategory,
  changePage: listChangePage,
  clearSearch: listClearSearch,
  onSearch: listOnSearch,
  initialize: listInitialize,
  cleanup: listCleanup,
} = useProductList({ pageSize: 20, homeRouteName: 'home' })

const listProductGroups = useProductListGroups(listProducts, listCategoryMap)

// ==================== SEO ====================
const seoCategoryName = computed(() => {
  if (!listSelectedCategory.value) return ''
  const cat = listCategoryMap.value.get(listSelectedCategory.value)
  return cat ? getLocalizedText(cat.name) : ''
})
usePageSeo({
  canonicalPath: () => route.path,
  title: () => {
    if (route.name === 'category-products') {
      return seoCategoryName.value || t('nav.products')
    }
    if (route.name === 'products') return t('nav.products')
    return undefined
  },
})

// ==================== Card Mode ====================
const formatDate = (dateString: string) => {
  if (!dateString) return ''
  return new Date(dateString).toLocaleDateString()
}

const goToProduct = (slug: string) => {
  router.push(`/products/${slug}`)
}

const loadFeaturedProducts = async () => {
  featuredLoading.value = true
  featuredError.value = false
  try {
    const response = await productAPI.list({ page: 1, page_size: 15 })
    products.value = response.data.data || []
  } catch (error) {
    featuredError.value = true
    console.error('Failed to load products:', error)
  } finally {
    featuredLoading.value = false
  }
}

const loadLatestPosts = async () => {
  if (!latestSectionVisible.value) return
  try {
    const params: Record<string, unknown> = { page: 1, page_size: 3 }
    if (blogEnabled.value && !noticeEnabled.value) params.type = 'blog'
    if (!blogEnabled.value && noticeEnabled.value) params.type = 'notice'
    const response = await postAPI.list(params)
    posts.value = response.data.data || []
  } catch (error) {
    console.error('Failed to load posts:', error)
  }
}

// ==================== Lifecycle ====================
const showAnnouncementIfNeeded = () => {
  const announcement = appStore.config?.announcement as HomeAnnouncement | undefined
  if (announcement && shouldShow(announcement)) {
    activeAnnouncement.value = announcement
    announcementVisible.value = true
  }
}

watch(isHome, (home) => {
  if (home) void recordHomeVisit()
}, { immediate: true })

onMounted(async () => {
  await appStore.loadConfig()
  if (templateMode.value === 'list') {
    await Promise.all([loadBanners(), listInitialize()])
  } else {
    await Promise.all([loadBanners(), loadFeaturedProducts(), loadLatestPosts()])
  }
  showAnnouncementIfNeeded()
})

onUnmounted(() => {
  stopHeroAutoPlay()
  listCleanup()
})
</script>

<style scoped>
.home-page {
  background: radial-gradient(ellipse 55rem 30rem at 76% 10rem, var(--ui-accent-soft), transparent),
    var(--ui-bg-page);
}

.home-hero {
  --hero-ink: #f5f8ff;
  --hero-muted: #c7d7f2;
  --hero-surface: #0a1530;
  --hero-spectrum: #77ebff, #7174ff, #d98afe, #ffac8f, #8cf1dc, #77ebff;
  position: relative;
  isolation: isolate;
  display: grid;
  align-content: center;
  min-height: clamp(18rem, 32vw, 26rem);
  overflow: hidden;
  border: 1px solid #839eff52;
  border-radius: clamp(1rem, 2.6vw, 2rem);
  background: var(--hero-surface);
  color: var(--hero-ink);
  box-shadow: 0 20px 48px -30px #365be995;
}

.home-hero-light,
.home-banner-image,
.home-hero-scrim {
  position: absolute;
  inset: 0;
  height: 100%;
  width: 100%;
  pointer-events: none;
}

.home-hero-light {
  overflow: hidden;
  background:
    radial-gradient(ellipse at 80% 62%, #422a8f 0%, transparent 49%),
    radial-gradient(ellipse at 95% 0%, #155385 0%, transparent 50%);
}

.home-orbit {
  position: absolute;
  top: 50%;
  left: 75%;
  width: clamp(21rem, 40vw, 38rem);
  aspect-ratio: 1;
  border-radius: 50%;
  transform: translate(-50%, -50%) rotate(-26deg) scaleY(.54);
  box-shadow: 0 0 36px #7d8fff80, inset 0 0 32px #697dff61;
}

.home-orbit::before,
.home-orbit::after {
  position: absolute;
  border-radius: inherit;
  content: '';
}

.home-orbit::before {
  inset: -3px;
  padding: 5px;
  background: conic-gradient(from 34deg, var(--hero-spectrum));
  mask: linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0);
  mask-composite: exclude;
  animation: home-spectrum 4.8s cubic-bezier(.24, .62, .38, 1) both;
}

.home-orbit::after {
  inset: 7%;
  border: 1px solid #b4c4ff50;
}

.home-hero--image .home-orbit {
  display: none;
}

.home-hero--image::after {
  position: absolute;
  z-index: 3;
  inset: auto 0 0;
  height: 2px;
  background: linear-gradient(90deg, var(--hero-spectrum));
  box-shadow: 0 0 12px #8bb4ff;
  pointer-events: none;
  content: '';
  animation: home-banner-spectrum 1.2s var(--ui-ease-out) both;
}

.home-banner-image {
  object-fit: cover;
}

.home-hero-scrim {
  background: linear-gradient(90deg, #071021f2 0%, #0a152ce6 32%, #0b16399c 68%, #0b16393d);
}

.home-hero-content {
  position: relative;
  padding: clamp(2rem, 5.8vw, 5rem);
}

.home-hero-title {
  max-width: 14ch;
  font-size: clamp(2.25rem, 6vw, 5rem);
  font-weight: 760;
  line-height: 1.05;
  letter-spacing: -.046em;
  overflow-wrap: anywhere;
  text-wrap: balance;
}

.home-hero-subtitle {
  max-width: 36ch;
  margin-top: clamp(.8rem, 2vw, 1.25rem);
  color: var(--hero-muted);
  font-size: clamp(1rem, 1.55vw, 1.25rem);
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.home-hero-skeleton {
  background: #b3c9f52e;
}

.home-hero-primary {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: space-between;
  gap: .75rem;
  padding: .625rem 1rem;
  border-radius: var(--ui-radius-sm);
  background: #c8f5ff;
  color: #0a2541;
  font-size: .95rem;
  font-weight: 700;
  transition: background-color .24s var(--ui-ease-out), scale .24s var(--ui-ease-out);
}

.home-hero-primary:hover { background: #edfcff; }
.home-hero-primary:active { scale: .97; }

.home-banner-controls {
  position: relative;
  justify-self: start;
  display: flex;
  align-items: center;
  max-width: calc(100% - 2.5rem);
  margin: -1rem clamp(2rem, 5.8vw, 5rem) clamp(1rem, 3vw, 2rem);
  border-radius: var(--ui-radius-sm);
  background: #091531df;
}

.home-banner-controls button {
  display: grid;
  min-width: 2.75rem;
  min-height: 2.75rem;
  place-items: center;
  border-radius: var(--ui-radius-sm);
  transition: background-color .24s var(--ui-ease-out);
}

.home-banner-controls button:hover { background: #9eb9ff26; }

.home-banner-controls .home-banner-step {
  width: 2.75rem;
  min-width: 0;
}

.home-banner-step::before {
  width: .4rem;
  height: .4rem;
  border-radius: 99px;
  background: #a4b5db;
  content: '';
  transition: transform .24s var(--ui-ease-out), background-color .24s var(--ui-ease-out);
}

.home-banner-step[aria-current='true']::before {
  background: #98efff;
  transform: scaleX(2.5);
}

.home-hero :is(a, button):focus-visible {
  outline: 2px solid #ecf7ff;
  outline-offset: 3px;
}

.home-catalog {
  scroll-margin-top: 5.5rem;
}

.home-catalog-heading { border: 0; padding-bottom: 0; }

.home-catalog :deep(.theme-price-sm) { font-family: inherit; }

@media (hover: hover) and (prefers-reduced-motion: no-preference) {
  .home-product {
    transition: translate .25s var(--ui-ease-out), border-color .25s var(--ui-ease-out), box-shadow .25s var(--ui-ease-out);
  }
  .home-product:hover { translate: 0 -3px; }
}

.home-catalog-link,
.home-post {
  transition: background-color .24s var(--ui-ease-out), border-color .24s var(--ui-ease-out);
}

.home-catalog-link:hover { background: var(--ui-accent-soft); }

.home-post { position: relative; }

.home-post:is(:hover, :focus-within) {
  border-color: var(--ui-accent);
}

.home-empty {
  background: var(--ui-bg-elevated);
}

.banner-fade-enter-active,
.banner-fade-leave-active {
  transition: opacity .4s var(--ui-ease-out);
}

.banner-fade-enter-active { z-index: 1; }
.home-hero-scrim { z-index: 2; }
.home-hero-content,
.home-banner-controls { z-index: 3; }
.banner-fade-enter-from,
.banner-fade-leave-to { opacity: 0; }

@keyframes home-spectrum {
  from { transform: rotate(-110deg); }
  to { transform: rotate(0); }
}

@keyframes home-banner-spectrum {
  from { transform: translateX(-45%); opacity: .2; }
  to { transform: translateX(0); opacity: 1; }
}

@media (max-width: 47.99rem) {
  .home-hero-light { opacity: .53; }
  .home-orbit { top: 28%; left: 97%; }
  .home-hero-content { padding-inline: 1.25rem; }
  .home-banner-controls { margin-inline: 1.25rem; }
  .home-hero-scrim { background: linear-gradient(90deg, #071021f2, #0b1639b8); }
}

@media (prefers-reduced-motion: reduce) {
  .home-orbit::before,
  .home-hero--image::after { animation: none; }
  .home-hero-primary:active { scale: 1; }
  .home-hero :is(a, button),
  .home-banner-step::before,
  .banner-fade-enter-active,
  .banner-fade-leave-active { transition: none; }
}
</style>
