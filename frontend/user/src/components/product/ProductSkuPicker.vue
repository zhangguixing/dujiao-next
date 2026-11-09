<template>
  <fieldset class="sku-picker">
    <legend class="mb-3 text-sm font-semibold text-foreground">
      {{ t('productDetail.skuTitle') }}
    </legend>
    <div class="sku-options">
      <label v-for="option in options" :key="option.id" class="relative min-w-0">
        <input
          v-model="selectedId"
          class="sku-input sr-only"
          type="radio"
          :name="groupName"
          :value="option.id"
          :disabled="option.disabled"
        />
        <span class="sku-option">
          <span class="min-w-0">
            <span class="block text-sm font-semibold leading-snug [overflow-wrap:anywhere]">{{ option.label }}</span>
            <span class="mt-1 block text-xs leading-snug text-ink-secondary">{{ option.stock }}</span>
          </span>
          <span class="sku-indicator" aria-hidden="true">
            <Check :size="12" :stroke-width="3" />
          </span>
        </span>
      </label>
    </div>
  </fieldset>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app'
import { resolveSkuStockDisplay, type PublicStockDisplay } from '@/utils/publicStock'
import { buildSkuDisplayText, normalizeSkuId } from '@/utils/sku'

interface SkuOption {
  id: number
  spec_values?: unknown
}

const props = defineProps<{
  product: object
  skus: SkuOption[]
  isPurchasable: (sku: SkuOption) => boolean
}>()

const selectedId = defineModel<number>({ required: true })
const groupName = useId()
const { t } = useI18n()
const appStore = useAppStore()

const stockText = (display: PublicStockDisplay) => {
  switch (display.kind) {
    case 'remaining':
      return t('productDetail.skuStockRemaining', { count: display.count })
    case 'range':
      return t('productDetail.skuStockRange', { min: display.min, max: display.max })
    case 'range_plus':
      return t('productDetail.skuStockRangePlus', { min: display.min })
    case 'unlimited':
      return t('productDetail.skuStockUnlimited')
    case 'out':
      return t('productDetail.skuStockOut')
    case 'low_stock':
      return t('productDetail.skuStockLow')
    case 'hidden':
      return t('productDetail.skuStockHidden')
    case 'in_stock':
      return t('productDetail.skuStockInStock')
  }
}

const options = computed(() => props.skus.map((sku) => ({
  id: normalizeSkuId(sku.id),
  label: buildSkuDisplayText({
    specValues: sku.spec_values,
    fallback: t('productDetail.skuFallback'),
    locale: appStore.locale,
  }),
  stock: stockText(resolveSkuStockDisplay(props.product, sku)),
  disabled: !props.isPurchasable(sku),
})))
</script>

<style scoped>
/* finesse · component: product SKU picker · register=commerce · tokens: storefront */
.sku-picker {
  container-type: inline-size;
  min-inline-size: 0;
}

.sku-options {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0.5rem;
}

@container (min-width: 26rem) {
  .sku-options {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.sku-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 3.5rem;
  height: 100%;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--ui-border);
  border-radius: var(--ui-radius-lg);
  background: var(--ui-bg-elevated);
  cursor: pointer;
}

.sku-indicator {
  display: grid;
  flex: none;
  width: 1.125rem;
  height: 1.125rem;
  place-items: center;
  border: 1px solid var(--ui-border-strong);
  border-radius: 50%;
  color: transparent;
}

@media (hover: hover) and (pointer: fine) {
  .sku-input:not(:checked, :disabled) + .sku-option:hover {
    border-color: var(--ui-border-strong);
    background: var(--ui-bg-soft);
  }
}

.sku-input:not(:disabled) + .sku-option:active {
  background: var(--ui-bg-muted);
}

.sku-input:focus-visible + .sku-option {
  outline: 2px solid var(--ui-accent);
  outline-offset: 2px;
}

.sku-input:checked:not(:disabled) + .sku-option {
  border-color: var(--ui-accent);
  background: var(--ui-accent-soft);
}

.sku-input:checked:not(:disabled) + .sku-option .sku-indicator {
  border-color: var(--ui-accent);
  background: var(--ui-accent);
  color: var(--ui-text-on-accent);
}

.sku-input:disabled + .sku-option {
  border-style: dashed;
  background: var(--ui-bg-soft);
  color: var(--ui-text-secondary);
  cursor: not-allowed;
}

.sku-input:disabled + .sku-option .sku-indicator {
  opacity: 0.5;
}
</style>
