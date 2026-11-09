<template>
  <div class="storefront-page min-h-screen">
    <div class="storefront-container">
      <!-- Page Header -->
      <header class="storefront-page-heading">
        <h1 class="storefront-page-title">{{ seoCategoryName || t('nav.products') }}</h1>
        <p class="storefront-page-subtitle">
          {{ t('products.subtitle') }}
        </p>
      </header>

      <div class="flex flex-col gap-4 lg:flex-row lg:gap-7">
        <CategorySidebar
          :categories="categoryGroups"
          :selected-category="selectedCategory"
          :expanded-parent-ids="expandedParentIds"
          :show-drawer="showFilterDrawer"
          @select-category="selectCategory"
          @toggle-parent="toggleParentCategory"
          @update:show-drawer="showFilterDrawer = $event"
        />

        <!-- Main Content - Products Grid -->
        <section class="min-w-0 flex-1" :aria-label="t('nav.products')" :aria-busy="loading">
          <div class="mb-4 flex flex-wrap items-center gap-3">
            <div class="relative min-w-0 flex-1 basis-52">
              <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <Input
                v-model="searchQuery"
                type="search"
                class="h-11 rounded-xl bg-card pl-10"
                :aria-label="t('products.searchLabel')"
                :placeholder="t('products.searchPlaceholder')"
              />
            </div>
            <Button v-if="searchQuery || selectedCategory" type="button" variant="outline" class="h-11" @click="clearSearch(); selectCategory(null)">
              {{ t('products.clearFilters') }}
            </Button>
          </div>
          <p class="mb-3 min-h-5 text-sm text-ink-secondary tabular-nums" aria-live="polite">
            {{ loading ? t('common.loading') : loadError ? '' : t('products.resultsCount', totalProducts) }}
          </p>
          <!-- Loading Skeleton -->
          <div v-if="loading" class="grid grid-cols-2 gap-3 md:gap-4 md:grid-cols-3 xl:grid-cols-4">
            <div v-for="i in 6" :key="i"
              class="rounded-2xl border bg-card overflow-hidden flex flex-col">
              <div class="h-36 md:h-56 theme-skeleton"></div>
              <div class="p-3 md:p-5 space-y-3">
                <div class="h-3 w-16 rounded theme-skeleton"></div>
                <div class="h-5 w-3/4 rounded theme-skeleton"></div>
                <div class="flex gap-2">
                  <div class="h-5 w-14 rounded-full theme-skeleton"></div>
                  <div class="h-5 w-14 rounded-full theme-skeleton"></div>
                </div>
                <div class="h-3 w-full rounded theme-skeleton"></div>
                <div class="h-3 w-2/3 rounded theme-skeleton"></div>
                <div class="border-t pt-3 flex justify-between items-center">
                  <div class="h-6 w-20 rounded theme-skeleton"></div>
                  <div class="h-4 w-16 rounded theme-skeleton"></div>
                </div>
              </div>
            </div>
          </div>

          <EmptyState v-else-if="loadError" size="lg" icon="alert" :title="t('common.api.requestFailed')">
            <template #action>
              <Button variant="secondary" @click="loadProducts">{{ t('errorBoundary.retry') }}</Button>
            </template>
          </EmptyState>

          <!-- Products Grid -->
          <div v-else-if="products.length > 0">
            <div class="grid grid-cols-2 gap-3 md:gap-4 md:grid-cols-3 xl:grid-cols-4">
              <ProductCard
                v-for="(product, idx) in products"
                :key="product.id"
                :product="product"
                :index="idx"
                :max-tags="isMobileGrid ? 1 : 2"
                :animation-step="50"
                @click="goToProduct"
                @quick-buy="openQuickBuy"
              />
            </div>

            <PaginationNav
              :current-page="currentPage"
              :total-pages="totalPages"
              :loading="loading"
              @change-page="changePage"
            />
          </div>

          <!-- Empty State -->
          <EmptyState
            v-else
            variant="soft"
            size="lg"
            :icon="(searchQuery || selectedCategory) ? 'search' : 'package'"
            :title="(searchQuery || selectedCategory) ? t('products.emptyFiltered') : t('products.empty')"
          >
            <template v-if="searchQuery || selectedCategory" #action>
              <Button variant="secondary" @click="clearSearch(); selectCategory(null)">
                {{ t('products.clearFilters') }}
              </Button>
            </template>
          </EmptyState>
        </section>
      </div>
    </div>

    <ProductQuickBuy
      v-if="quickBuyProduct"
      :product="quickBuyProduct"
      :visible="quickBuyVisible"
      @update:visible="quickBuyVisible = $event"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Search } from 'lucide-vue-next'
import { useProductList } from '../composables/useProductList'
import { usePageSeo } from '../composables/usePageSeo'
import { useLocalized } from '../composables/useProduct'
import ProductCard from '../components/ProductCard.vue'
import ProductQuickBuy from '../components/ProductQuickBuy.vue'
import CategorySidebar from '../components/CategorySidebar.vue'
import PaginationNav from '../components/PaginationNav.vue'
import EmptyState from '../components/EmptyState.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

const router = useRouter()
const { t } = useI18n()

const {
  loading,
  loadError,
  products,
  selectedCategory,
  searchQuery,
  currentPage,
  totalPages,
  totalProducts,
  showFilterDrawer,
  expandedParentIds,
  categoryGroups,
  categoryMap,
  selectCategory,
  toggleParentCategory,
  changePage,
  loadProducts,
  clearSearch,
  initialize,
  cleanup,
} = useProductList({ pageSize: 12, homeRouteName: 'products' })

// ==================== SEO ====================
const route = useRoute()
const { getLocalizedText } = useLocalized()
const seoCategoryName = computed(() => {
  if (!selectedCategory.value) return ''
  const cat = categoryMap.value.get(selectedCategory.value)
  return cat ? getLocalizedText(cat.name) : ''
})
usePageSeo({
  canonicalPath: () => route.path,
  title: () => {
    if (route.name === 'category-products') {
      return seoCategoryName.value || t('nav.products')
    }
    return t('nav.products')
  },
})

const quickBuyProduct = ref<any>(null)
const quickBuyVisible = ref(false)

const openQuickBuy = (product: any) => {
  quickBuyProduct.value = product
  quickBuyVisible.value = true
}

// Detect mobile 2-col grid (< md breakpoint)
const isMobileGrid = ref(window.innerWidth < 768)
const handleResize = () => {
  isMobileGrid.value = window.innerWidth < 768
}

const goToProduct = (slug: string) => {
  router.push(`/products/${slug}`)
}

onMounted(async () => {
  window.addEventListener('resize', handleResize, { passive: true })
  await initialize()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  cleanup()
})
</script>

<style scoped>
.line-clamp-1 {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
  line-clamp: 1;
}
.line-clamp-2 {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
}
</style>
