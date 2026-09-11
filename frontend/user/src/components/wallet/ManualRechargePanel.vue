<template>
  <section v-if="channels.length || activeRequest" class="rounded-2xl border bg-card p-5 shadow-sm">
    <div class="mb-5 flex items-start justify-between gap-4">
      <div><h2 class="text-lg font-bold text-foreground">扫码人工充值</h2><p class="mt-1 text-sm text-muted-foreground">转账后上传付款截图并填写流水号，审核通过后自动到账。</p></div>
      <Badge v-if="activeRequest" variant="secondary">有待处理申请</Badge>
    </div>
    <div v-if="activeRequest" class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
      <div class="font-medium">申请 {{ activeRequest.request_no }} 正在{{ statusLabel(activeRequest.status) }}</div>
      <div class="mt-1">金额：{{ activeRequest.amount }} {{ activeRequest.currency }}</div>
      <Button v-if="activeRequest.status === 'pending'" class="mt-3" size="sm" variant="outline" @click="cancel">撤销申请</Button>
    </div>
    <form v-else class="grid gap-4 md:grid-cols-2" @submit.prevent="submit">
      <div class="space-y-2"><Label>收款方式</Label><Select v-model="form.channelId"><SelectTrigger><SelectValue placeholder="选择微信或支付宝" /></SelectTrigger><SelectContent><SelectItem v-for="item in channels" :key="item.id" :value="String(item.id)">{{ item.name }}</SelectItem></SelectContent></Select></div>
      <div class="space-y-2"><Label>充值金额</Label><Input v-model="form.amount" inputmode="decimal" placeholder="例如 100.00" /></div>
      <div v-if="selected" class="md:col-span-2 rounded-xl border bg-muted/30 p-4"><div class="grid gap-4 sm:grid-cols-[160px_1fr]"><img :src="selected.qr_code_url" alt="收款二维码" class="h-40 w-40 rounded-lg border bg-white object-contain p-2" /><div><div class="font-semibold">{{ selected.account_name || selected.name }}</div><p class="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">{{ selected.instructions || '请按填写金额完成转账。' }}</p><p v-if="selected.min_amount !== '0.00' || selected.max_amount !== '0.00'" class="mt-2 text-xs text-muted-foreground">金额范围：{{ selected.min_amount }} - {{ selected.max_amount || '不限' }}</p></div></div></div>
      <div class="space-y-2"><Label>交易流水号</Label><Input v-model="form.transactionNo" placeholder="请从支付账单复制" /></div>
      <div class="space-y-2"><Label>联系方式</Label><div class="flex gap-2"><Select v-model="form.contactType"><SelectTrigger class="w-24"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="email">邮箱</SelectItem><SelectItem value="tg">TG</SelectItem></SelectContent></Select><Input v-model="form.contactValue" placeholder="用于审核沟通" /></div></div>
      <div class="space-y-2 md:col-span-2"><Label>付款截图 <span class="text-destructive">*</span></Label><Input type="file" accept="image/png,image/jpeg,image/webp" :disabled="uploading" @change="upload" /><p v-if="proofURL" class="text-xs text-emerald-600">截图已上传</p></div>
      <div v-if="error" class="md:col-span-2 rounded-lg bg-destructive/10 p-3 text-sm text-destructive">{{ error }}</div>
      <div class="md:col-span-2"><Button type="submit" :disabled="submitting || uploading">{{ submitting ? '正在提交…' : '提交充值申请' }}</Button></div>
    </form>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { walletAPI } from '@/api'
import { Button } from '@/components/ui/button'; import { Input } from '@/components/ui/input'; import { Label } from '@/components/ui/label'; import { Badge } from '@/components/ui/badge'; import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
const emit = defineEmits<{ refresh: [] }>(); const channels = ref<any[]>([]); const activeRequest = ref<any>(null); const uploading = ref(false); const submitting = ref(false); const proofURL = ref(''); const error = ref('')
const form = reactive({ channelId: '', amount: '', transactionNo: '', contactType: 'email', contactValue: '' }); const selected = computed(() => channels.value.find(item => String(item.id) === form.channelId))
const statusLabel = (value: string) => ({ pending: '等待审核', processing: '核验中' } as any)[value] || value
async function load() { const [channelRes, requestRes] = await Promise.all([walletAPI.manualRechargeChannels(), walletAPI.manualRecharges({ page: 1, page_size: 20 })]); channels.value = channelRes.data.data || []; activeRequest.value = (requestRes.data.data || []).find((item: any) => ['pending', 'processing'].includes(item.status)) || null }
async function upload(event: Event) { const file = (event.target as HTMLInputElement).files?.[0]; if (!file) return; error.value = ''; uploading.value = true; try { const data = new FormData(); data.append('file', file); const response = await walletAPI.uploadManualRechargeProof(data); proofURL.value = response.data.data?.url || '' } catch (err: any) { error.value = err?.message || '截图上传失败' } finally { uploading.value = false } }
async function submit() { error.value = ''; if (!form.channelId || !form.amount || !form.transactionNo || !form.contactValue || !proofURL.value) { error.value = '请完整填写并上传付款截图'; return }; submitting.value = true; try { activeRequest.value = (await walletAPI.createManualRecharge({ channel_id: Number(form.channelId), amount: form.amount, transaction_no: form.transactionNo, contact_type: form.contactType, contact_value: form.contactValue, proof_url: proofURL.value })).data.data; emit('refresh') } catch (err: any) { error.value = err?.message || '提交失败' } finally { submitting.value = false } }
async function cancel() { try { await walletAPI.cancelManualRecharge(activeRequest.value.request_no); activeRequest.value = null; await load() } catch (err: any) { error.value = err?.message || '撤销失败' } }
onMounted(() => { void load() })
</script>
