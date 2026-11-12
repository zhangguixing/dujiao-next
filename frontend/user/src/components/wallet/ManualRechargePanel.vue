<template>
  <section v-if="available || activeRequest" class="rounded-2xl border bg-card p-5 shadow-sm">
    <div class="mb-5 flex items-start justify-between gap-4">
      <div>
        <h2 class="text-lg font-bold text-foreground">{{ t('personalCenter.wallet.manualRecharge.title') }}</h2>
        <p v-if="available && channels.length" class="mt-1 text-sm text-muted-foreground">
          {{ t('personalCenter.wallet.manualRecharge.description') }}
        </p>
      </div>
      <Badge v-if="activeRequest" variant="secondary">{{ t('personalCenter.wallet.manualRecharge.active') }}</Badge>
    </div>
    <div v-if="activeRequest" class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
      <div class="font-medium">{{ t('personalCenter.wallet.manualRecharge.status.' + activeRequest.status, { number: activeRequest.request_no }) }}</div>
      <div class="mt-1">{{ t('personalCenter.wallet.amountLabel') }} · {{ activeRequest.amount }} {{ activeRequest.currency }}</div>
      <Button v-if="activeRequest.status === 'pending'" class="mt-3" size="sm" variant="outline" @click="cancel">
        {{ t('personalCenter.wallet.manualRecharge.cancel') }}
      </Button>
    </div>
    <p v-else-if="loading" role="status" class="text-sm text-muted-foreground">{{ t('common.loading') }}</p>
    <div v-else-if="loadFailed" class="flex flex-wrap items-center gap-3">
      <p role="alert" class="text-sm text-muted-foreground">{{ t('personalCenter.wallet.manualRecharge.loadFailed') }}</p>
      <Button size="sm" variant="outline" @click="load">{{ t('errorBoundary.retry') }}</Button>
    </div>
    <p v-else-if="!channels.length" role="status" class="text-sm text-muted-foreground">
      {{ t('personalCenter.wallet.manualRecharge.noChannels') }}
    </p>
    <form v-else class="grid gap-4 md:grid-cols-2" @submit.prevent="submit">
      <div class="space-y-2">
        <Label>{{ t('personalCenter.wallet.manualRecharge.channel') }}</Label>
        <Select v-model="form.channelId">
          <SelectTrigger><SelectValue :placeholder="t('personalCenter.wallet.channelPlaceholder')" /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="item in channels" :key="item.id" :value="String(item.id)">{{ item.name }}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="space-y-2">
        <Label>{{ t('personalCenter.wallet.amountLabel') }}</Label>
        <Input v-model="form.amount" inputmode="decimal" :placeholder="t('personalCenter.wallet.amountPlaceholder')" />
      </div>
      <div v-if="selected" class="md:col-span-2 rounded-xl border bg-muted/30 p-4">
        <div class="grid gap-4 sm:grid-cols-[160px_1fr]">
          <img :src="selected.qr_code_url" :alt="t('personalCenter.wallet.manualRecharge.qrCode')" class="h-40 w-40 rounded-lg border bg-white object-contain p-2" />
          <div>
            <div class="font-semibold">{{ selected.account_name || selected.name }}</div>
            <p class="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
              {{ selected.instructions || t('personalCenter.wallet.manualRecharge.instructions') }}
            </p>
            <p v-if="selected.min_amount !== '0.00' || selected.max_amount !== '0.00'" class="mt-2 text-xs text-muted-foreground">
              {{ t('personalCenter.wallet.manualRecharge.amountRange', { min: selected.min_amount, max: selected.max_amount || t('personalCenter.wallet.manualRecharge.unlimited') }) }}
            </p>
          </div>
        </div>
      </div>
      <div class="space-y-2">
        <Label>{{ t('personalCenter.wallet.manualRecharge.transactionNo') }}</Label>
        <Input v-model="form.transactionNo" :placeholder="t('personalCenter.wallet.manualRecharge.transactionPlaceholder')" />
      </div>
      <div class="space-y-2">
        <Label>{{ t('personalCenter.wallet.manualRecharge.contact') }}</Label>
        <div class="flex gap-2">
          <Select v-model="form.contactType">
            <SelectTrigger class="w-24"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="email">{{ t('personalCenter.wallet.manualRecharge.email') }}</SelectItem>
              <SelectItem value="tg">Telegram</SelectItem>
            </SelectContent>
          </Select>
          <Input v-model="form.contactValue" :placeholder="t('personalCenter.wallet.manualRecharge.contactPlaceholder')" />
        </div>
      </div>
      <div class="space-y-2 md:col-span-2">
        <Label>{{ t('personalCenter.wallet.manualRecharge.proof') }} <span class="text-destructive">*</span></Label>
        <Input type="file" accept="image/png,image/jpeg,image/webp" :disabled="uploading" @change="upload" />
        <p v-if="proofURL" class="text-xs text-emerald-600">{{ t('personalCenter.wallet.manualRecharge.proofUploaded') }}</p>
      </div>
      <div v-if="error" role="alert" class="md:col-span-2 rounded-lg bg-destructive/10 p-3 text-sm text-destructive">{{ error }}</div>
      <div class="md:col-span-2">
        <Button type="submit" :disabled="submitting || uploading">
          {{ t(submitting ? 'personalCenter.wallet.recharging' : 'personalCenter.wallet.manualRecharge.submit') }}
        </Button>
      </div>
    </form>
    <p v-if="activeRequest && error" role="alert" class="mt-3 text-sm text-destructive">{{ error }}</p>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { walletAPI } from '@/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
const props = defineProps<{ enabled?: boolean }>()
const emit = defineEmits<{ refresh: [] }>()
const { t } = useI18n()
const channels = ref<any[]>([])
const activeRequest = ref<any>(null)
const available = computed(() => props.enabled ?? (channels.value.length > 0))
const loading = ref(true)
const loadFailed = ref(false)
const uploading = ref(false)
const submitting = ref(false)
const proofURL = ref('')
const error = ref('')
const form = reactive({ channelId: '', amount: '', transactionNo: '', contactType: 'email', contactValue: '' })
const selected = computed(() => channels.value.find(item => String(item.id) === form.channelId))

async function load() {
  loading.value = true
  loadFailed.value = false
  const [channelResult, requestResult] = await Promise.allSettled([
    walletAPI.manualRechargeChannels(),
    walletAPI.manualRecharges({ page: 1, page_size: 20 }),
  ])
  if (channelResult.status === 'fulfilled') {
    channels.value = channelResult.value.data.data || []
  }
  if (requestResult.status === 'fulfilled') {
    activeRequest.value = (requestResult.value.data.data || []).find((item: any) => ['pending', 'processing'].includes(item.status)) || null
  }
  loadFailed.value = channelResult.status === 'rejected' || requestResult.status === 'rejected'
  loading.value = false
}

async function upload(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  error.value = ''
  uploading.value = true
  try {
    const data = new FormData()
    data.append('file', file)
    const response = await walletAPI.uploadManualRechargeProof(data)
    proofURL.value = response.data.data?.url || ''
  } catch (err: any) {
    error.value = err?.message || t('personalCenter.wallet.manualRecharge.uploadFailed')
  } finally {
    uploading.value = false
  }
}

async function submit() {
  error.value = ''
  if (!form.channelId || !form.amount || !form.transactionNo || !form.contactValue || !proofURL.value) {
    error.value = t('personalCenter.wallet.manualRecharge.incomplete')
    return
  }
  submitting.value = true
  try {
    const response = await walletAPI.createManualRecharge({
      channel_id: Number(form.channelId),
      amount: form.amount,
      transaction_no: form.transactionNo,
      contact_type: form.contactType,
      contact_value: form.contactValue,
      proof_url: proofURL.value,
    })
    activeRequest.value = response.data.data
    emit('refresh')
  } catch (err: any) {
    error.value = err?.message || t('personalCenter.wallet.errors.rechargeFailed')
  } finally {
    submitting.value = false
  }
}

async function cancel() {
  error.value = ''
  try {
    await walletAPI.cancelManualRecharge(activeRequest.value.request_no)
    activeRequest.value = null
    await load()
  } catch (err: any) {
    error.value = err?.message || t('personalCenter.wallet.manualRecharge.cancelFailed')
  }
}

onMounted(() => { void load() })
</script>
