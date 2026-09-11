import { userApi } from './client'
import type {
    WalletRechargePayload,
    CaptchaPayload,
} from './types'

export const walletAPI = {
    getPaymentChannels: (amount: string) => userApi.post('/wallet/payment-channels', { amount }),
    account: () => userApi.get('/wallet'),
    transactions: (params?: any) => userApi.get('/wallet/transactions', { params }),
    recharge: (data: WalletRechargePayload) => userApi.post('/wallet/recharge', data),
    rechargeOrders: (params?: any) => userApi.get('/wallet/recharges', { params }),
    rechargeStats: (params?: any) => userApi.get('/wallet/recharges/stats', { params }),
    rechargeDetail: (rechargeNo: string) =>
        userApi.get(`/wallet/recharges/${encodeURIComponent(rechargeNo)}`),
    captureRechargePayment: (paymentID: number) =>
        userApi.post(`/wallet/recharge/payments/${paymentID}/capture`),
    manualRechargeChannels: () => userApi.get('/wallet/manual-recharge-channels'),
    createManualRecharge: (data: any) => userApi.post('/wallet/manual-recharges', data),
    manualRecharges: (params?: any) => userApi.get('/wallet/manual-recharges', { params }),
    cancelManualRecharge: (requestNo: string) => userApi.post(`/wallet/manual-recharges/${encodeURIComponent(requestNo)}/cancel`, {}),
    uploadManualRechargeProof: (data: FormData) => userApi.post('/uploads/manual-recharge-proof', data),
}

export const giftCardAPI = {
    redeem: (data: { code: string; captcha_payload?: CaptchaPayload }) =>
        userApi.post('/gift-cards/redeem', data),
}
