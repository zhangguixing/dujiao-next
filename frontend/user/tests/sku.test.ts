import test from 'node:test'
import assert from 'node:assert/strict'
import { buildSkuDisplayTextFromSnapshot, formatSkuSpecValues } from '../src/utils/sku.ts'

test('imported SKU translations display a single label in the storefront locale', () => {
  const specValues = {
    'en-US': 'gpt go',
    'ru-RU': 'gpt go',
    'tr-TR': 'gpt go',
    'vi-VN': 'gpt go',
    'zh-CN': 'gpt go',
    'zh-TW': 'gpt go',
  }

  assert.equal(formatSkuSpecValues(specValues, 'zh-CN'), 'gpt go')
  assert.equal(buildSkuDisplayTextFromSnapshot({
    spec_values: { ...specValues, 'en-US': 'GPT Go (English)' },
  }, { locale: 'en-US' }), 'GPT Go (English)')
})

test('SKU labels fall back to an available translation while retaining named specs', () => {
  assert.equal(formatSkuSpecValues({ 'en-US': ' ', 'vi-VN': 'gpt go' }, 'zh-TW'), 'gpt go')
  assert.equal(formatSkuSpecValues({ color: { 'en-US': 'Black', 'ru-RU': 'Черный' }, capacity: '256 GB' }, 'en-US'), 'color:Black / capacity:256 GB')
})
