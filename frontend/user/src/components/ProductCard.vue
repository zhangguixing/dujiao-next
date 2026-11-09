<template>
  <Card
    class="group relative flex h-full min-w-0 flex-col overflow-hidden rounded-xl transition-[border-color,box-shadow] theme-slide-up"
    :class="isSoldOut(product)
      ? 'cursor-default opacity-85 grayscale-[0.25] saturate-50 border-destructive/30'
      : 'cursor-pointer hover:border-primary/40 hover:shadow-soft focus-within:border-primary'"
    :style="{ animationDelay: `${index * animationStep}ms` }"
    @click="$emit('click', product.slug)">
    <!-- Image Area -->
    <div class="aspect-[4/3] overflow-hidden bg-muted relative shrink-0">
      <img v-if="displayImageSrc && !imageErrored" :src="displayImageSrc"
        :alt="getLocalizedText(product.title)" loading="lazy" decoding="async"
        class="w-full h-full object-cover transform transition-transform duration-700 ease-out"
        :class="[
          isSoldOut(product) ? 'grayscale brightness-75' : 'motion-safe:[@media(hover:hover)]:group-hover:scale-105',
        ]"
        @error="handleImageError" />
      <div v-else class="w-full h-full flex items-center justify-center text-muted-foreground" role="img"
        :aria-label="getLocalizedText(product.title)">
        <ImageIcon class="w-8 h-8 md:w-12 md:h-12" :stroke-width="1.5" aria-hidden="true" />
      </div>

      <div v-if="isSoldOut(product)" class="absolute inset-0 z-20 bg-black/45"></div>
      <Badge
        v-if="isSoldOut(product)"
        variant="destructive"
        size="xs"
        class="absolute left-2 top-2 md:left-4 md:top-4 z-30 tracking-wider shadow-sm"
      >
        {{ t('products.stockStatus.outOfStock') }}
      </Badge>

      <!-- Tags -->
      <div v-if="!isSoldOut(product) && product.tags && product.tags.length > 0"
        class="absolute top-2 right-2 md:top-4 md:right-4 z-20 flex flex-wrap gap-1 md:gap-2 justify-end">
        <span v-for="(tag, tagIndex) in product.tags.slice(0, maxTags)" :key="tagIndex"
          class="inline-flex items-center rounded-md border border-white/25 bg-black/55 px-2 md:px-3 py-0.5 md:py-1 text-xs font-medium text-white backdrop-blur-sm">
          {{ tag }}
        </span>
      </div>
    </div>

    <!-- Content Area -->
    <div class="relative z-20 flex flex-1 flex-col p-2.5 sm:p-3 md:p-4">
      <div v-if="product.category?.name" class="mb-1 truncate text-xs text-ink-secondary">
        {{ t('products.categoryLabel') }} · {{ getLocalizedText(product.category.name) }}
      </div>
      <h3 class="mb-2 text-sm font-semibold leading-snug text-foreground md:text-base">
        <router-link
          :to="`/products/${product.slug}`"
          class="line-clamp-2 rounded-sm [overflow-wrap:anywhere] hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          @click.stop
        >
          {{ getLocalizedText(product.title) }}
        </router-link>
      </h3>

      <!-- Badges -->
      <div class="mb-2 md:mb-3 flex flex-wrap items-center gap-1 md:gap-2">
        <!-- Mobile: show only fulfillment type badge -->
        <Badge
          class="md:hidden"
          size="xs"
          :variant="product.fulfillment_type === 'auto' ? 'info' : 'neutral'"
        >
          {{ getFulfillmentTypeLabel(product.fulfillment_type) }}
        </Badge>

        <!-- Desktop: show all badges -->
        <Badge
          class="hidden md:inline-flex"
          size="xs"
          :variant="product.purchase_type === 'guest' ? 'warning' : 'success'"
        >
          <UserPlus v-if="product.purchase_type === 'guest'" class="h-3 w-3" />
          <Lock v-else class="h-3 w-3" />
          {{ getPurchaseTypeLabel(product.purchase_type) }}
        </Badge>

        <Badge
          class="hidden md:inline-flex"
          size="xs"
          :variant="product.fulfillment_type === 'auto' ? 'info' : 'neutral'"
        >
          <Zap v-if="product.fulfillment_type === 'auto'" class="h-3 w-3" />
          <Pencil v-else class="h-3 w-3" />
          {{ getFulfillmentTypeLabel(product.fulfillment_type) }}
        </Badge>

        <Badge class="hidden md:inline-flex" size="xs" :variant="getStockBadgeVariant(product.stock_status)">
          {{ getStockStatusLabel(product) }}
        </Badge>
      </div>

      <p class="hidden md:block text-muted-foreground text-sm mb-6 line-clamp-2">
        {{ getLocalizedText(product.description) }}
      </p>

      <div class="mt-auto flex items-center justify-between gap-2 border-t pt-2.5 md:pt-3">
        <div class="flex min-w-0 flex-col">
          <span class="hidden md:block text-xs text-muted-foreground uppercase tracking-wider">{{ t('products.price') }}</span>
          <span
            v-if="hasPromotionPrice(product)"
            class="theme-price-sm theme-price-promotion text-sm [overflow-wrap:anywhere] md:text-lg"
            :aria-label="t('products.promotionPriceAria', { price: formatPrice(getPromotionPriceAmount(product), siteCurrency) })"
          >
            {{ formatPrice(getPromotionPriceAmount(product), siteCurrency) }}
          </span>
          <span
            v-else
            class="theme-price-sm text-sm [overflow-wrap:anywhere] md:text-lg"
            :aria-label="t('products.priceAria', { price: formatPrice(product.price_amount, siteCurrency) })"
          >
            {{ formatPrice(product.price_amount, siteCurrency) }}
          </span>
          <div v-if="hasPromotionPrice(product)" class="mt-0.5 flex flex-wrap items-center gap-1.5">
            <span
              class="hidden md:inline text-xs text-muted-foreground opacity-80 line-through"
              :aria-label="t('products.originalPriceAria', { price: formatPrice(product.price_amount, siteCurrency) })"
            >{{ formatPrice(product.price_amount, siteCurrency) }}</span>
            <Badge variant="danger" size="xs">
              {{ t('products.promotionTag') }}
            </Badge>
          </div>
          <div v-else-if="hasWholesalePrices(product)" class="mt-0.5 flex flex-wrap items-center gap-1.5">
            <Badge variant="success" size="xs">
              {{ t('products.wholesaleTag') }}
            </Badge>
          </div>
          <div v-else-if="hasPromotionRules(product)" class="mt-0.5 flex flex-wrap items-center gap-1.5">
            <Badge variant="warning" size="xs">
              {{ t('products.promotionBadge') }}
            </Badge>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-2">
          <!-- Quick buy cart button -->
          <Button
            type="button"
            variant="outline"
            size="icon"
            class="size-11 rounded-xl"
            :aria-label="t('products.quickBuyAria')"
            :disabled="isSoldOut(product)"
            @click.stop="$emit('quickBuy', product)"
          >
            <ShoppingCart class="h-4 w-4" />
          </Button>
          <!-- Desktop: view details -->
          <span
            class="hidden md:flex text-xs uppercase font-bold transition-colors items-center gap-1"
            :class="isSoldOut(product)
              ? 'text-destructive/90'
              : 'text-muted-foreground group-hover:text-foreground'">
            <ArrowRight class="w-4 h-4 transition-transform" :class="isSoldOut(product) ? '' : 'group-hover:translate-x-1'" />
          </span>
        </div>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import { ArrowRight, Image as ImageIcon, Lock, Pencil, ShoppingCart, UserPlus, Zap } from 'lucide-vue-next'
import { getFirstImageUrl, getImageUrl } from '../utils/image'
import { useLocalized, useProductLabels } from '../composables/useProduct'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

const props = withDefaults(defineProps<{
  product: any
  index?: number
  maxTags?: number
  animationStep?: number
}>(), {
  index: 0,
  maxTags: 2,
  animationStep: 50,
})

defineEmits<{
  click: [slug: string]
  quickBuy: [product: any]
}>()

const { t } = useI18n()
const { getLocalizedText, siteCurrency, formatPrice } = useLocalized()
const { getPurchaseTypeLabel, getFulfillmentTypeLabel, getStockBadgeVariant, getStockStatusLabel, isSoldOut, hasPromotionPrice, getPromotionPriceAmount, hasPromotionRules, hasWholesalePrices } = useProductLabels()

const imageErrored = ref(false)
const attemptIdx = ref(0)

const imageCandidates = computed<string[]>(() => {
  const arr: string[] = []
  const primary = getFirstImageUrl(props.product?.images)
  if (primary) arr.push(primary)
  const categoryIcon = props.product?.category?.icon
  if (categoryIcon) {
    const resolved = getImageUrl(categoryIcon)
    if (resolved && resolved !== primary) arr.push(resolved)
  }
  return arr
})

const displayImageSrc = computed(() => imageCandidates.value[attemptIdx.value] ?? '')

watch(imageCandidates, () => {
  attemptIdx.value = 0
  imageErrored.value = false
}, { deep: true })

const handleImageError = () => {
  if (attemptIdx.value < imageCandidates.value.length - 1) {
    attemptIdx.value++
  } else {
    imageErrored.value = true
  }
}
</script>
