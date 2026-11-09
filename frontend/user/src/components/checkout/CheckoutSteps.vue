<template>
  <ol
    class="storefront-surface flex list-none items-start p-3 sm:p-4"
    :aria-label="t('checkoutSteps.label')"
  >
    <template v-for="(step, idx) in steps" :key="step.key">
      <li
        class="relative flex min-w-0 flex-1 flex-col items-center gap-1.5"
        :aria-current="step.status === 'current' ? 'step' : undefined"
      >
        <span
          v-if="idx > 0"
          aria-hidden="true"
          class="absolute top-3.5 right-1/2 -left-1/2 h-px"
          :class="step.status !== 'upcoming' ? 'bg-primary' : 'bg-muted'"
        ></span>
        <div class="relative z-10 flex min-w-0 flex-col items-center gap-1.5">
          <span
            class="flex size-7 items-center justify-center rounded-full border text-xs font-semibold"
            aria-hidden="true"
            :class="step.status !== 'upcoming'
              ? 'bg-primary text-primary-foreground border-transparent'
              : 'border-border bg-card text-ink-secondary'"
          >
            <Check v-if="step.status === 'done'" class="w-3.5 h-3.5" :stroke-width="3" />
            <span v-else>{{ idx + 1 }}</span>
          </span>
          <span
            class="whitespace-nowrap text-center text-xs sm:text-sm"
            :class="step.status === 'current' ? 'font-semibold text-foreground' : 'text-ink-secondary'"
          >
            {{ step.label }}
          </span>
        </div>
      </li>
    </template>
  </ol>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check } from 'lucide-vue-next'

type StepKey = 'cart' | 'checkout' | 'payment'
type StepStatus = 'done' | 'current' | 'upcoming'

const props = withDefaults(
  defineProps<{
    currentStep: StepKey
    stepKeys?: StepKey[]
  }>(),
  {
    stepKeys: () => ['cart', 'checkout', 'payment'],
  },
)

const { t } = useI18n()

const steps = computed(() => {
  const activeIdx = props.stepKeys.indexOf(props.currentStep)
  if (activeIdx < 0 && import.meta.env.DEV) {
    const brandWarn = (globalThis as any).console?.warn?.bind(console)
    brandWarn?.(`[CheckoutSteps] currentStep "${props.currentStep}" not found in stepKeys`, props.stepKeys)
  }
  return props.stepKeys.map((key, idx) => {
    const status: StepStatus = activeIdx < 0
      ? 'upcoming'
      : idx < activeIdx
        ? 'done'
        : idx === activeIdx
          ? 'current'
          : 'upcoming'
    return { key, label: t(`${key}.title`), status }
  })
})
</script>
