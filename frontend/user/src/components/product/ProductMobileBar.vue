<template>
  <Transition
    enter-active-class="transition duration-300 ease-out"
    enter-from-class="translate-y-full opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="translate-y-full opacity-0">
    <div v-if="visible"
      class="storefront-purchase-bar lg:hidden fixed bottom-[calc(var(--storefront-tabbar-height)+var(--storefront-bottom-inset))] left-0 right-0 z-40 bg-card border-t">
      <div class="flex items-center gap-2 px-3 py-2">
        <!-- Price -->
        <div class="flex-1 min-w-0">
          <span v-if="showMemberPrice" class="theme-price-sm text-amber-600 dark:text-amber-300 truncate block">
            {{ memberPriceDisplay }}
          </span>
          <span v-else-if="showSkuPromotionPrice" class="theme-price-sm text-rose-600 dark:text-rose-300 truncate block">
            {{ skuPromotionPriceDisplay }}
          </span>
          <span v-else-if="showSkuPrice" class="theme-price-sm text-primary truncate block">
            {{ skuPriceDisplay }}
          </span>
          <span v-else-if="showProductPromotionPrice" class="theme-price-sm text-rose-600 dark:text-rose-300 truncate block">
            {{ productPromotionPriceDisplay }}
          </span>
          <span v-else class="theme-price-sm text-primary truncate block">
            {{ productPriceDisplay }}
          </span>
        </div>
        <!-- Actions -->
        <Button v-if="requiresLogin" size="lg" class="min-h-11 rounded-xl px-3 font-semibold" @click="$emit('goLogin')">
          {{ t('productDetail.loginToBuy') }}
        </Button>
        <template v-else>
          <Button variant="secondary" size="lg" class="min-h-11 rounded-xl px-3 font-semibold" :disabled="!canPurchase" @click="$emit('addToCart')">
            {{ t('productDetail.addToCart') }}
          </Button>
          <Button size="lg" class="min-h-11 rounded-xl px-3 font-semibold" :disabled="!canPurchase" @click="$emit('buyNow')">
            {{ t('productDetail.buyNow') }}
          </Button>
        </template>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'

const { t } = useI18n()

defineProps<{
  visible: boolean
  requiresLogin: boolean
  canPurchase: boolean
  showMemberPrice: boolean
  memberPriceDisplay: string
  showSkuPromotionPrice: boolean
  skuPromotionPriceDisplay: string
  showSkuPrice: boolean
  skuPriceDisplay: string
  showProductPromotionPrice: boolean
  productPromotionPriceDisplay: string
  productPriceDisplay: string
}>()

defineEmits<{
  addToCart: []
  buyNow: []
  goLogin: []
}>()
</script>
