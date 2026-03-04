import type { Model } from '@/lib/api'

export type ModelCategory = 'text' | 'image' | 'video' | 'audio'
export type ModelCategoryFilter = ModelCategory | 'all'

export function getModelCategory(model: Pick<Model, 'name' | 'provider'>): ModelCategory {
  const haystack = `${model.provider} ${model.name}`.toLowerCase()

  if (/(sora|kling|video|runway|luma|pika|gen-?3|hunyuan|veo)/.test(haystack)) return 'video'
  if (/(dall|dalle|stable|diffusion|sdxl|midjourney|flux|image|img|kolors|banana)/.test(haystack)) return 'image'
  if (/(whisper|tts|audio|voice|speech|asr|music)/.test(haystack)) return 'audio'

  return 'text'
}

export function getModelCategoryLabel(category: ModelCategoryFilter): string {
  switch (category) {
    case 'all':
      return '全部'
    case 'text':
      return '文本'
    case 'image':
      return '图片'
    case 'video':
      return '视频'
    case 'audio':
      return '音频'
    default:
      return '全部'
  }
}

export function formatPriceUnit(priceUnit: number): string {
  if (!Number.isFinite(priceUnit) || priceUnit <= 0) return 'tokens'
  if (priceUnit === 1000) return '1K tokens'
  if (priceUnit === 1_000_000) return '1M tokens'
  return `${priceUnit.toLocaleString('en-US')} tokens`
}

export function parseNumber(value: number | string | undefined | null): number | null {
  if (value === undefined || value === null) return null
  if (typeof value === 'number') return Number.isFinite(value) ? value : null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

export function formatCnyAmount(value: number | null, maximumFractionDigits = 6): string {
  if (value === null) return '—'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 0,
    maximumFractionDigits,
  }).format(value)
}

export function formatModelPricing(model: Pick<Model, 'input_price' | 'output_price' | 'price_unit'>): {
  input: string
  output: string
  unitLabel: string
} {
  const input = parseNumber(model.input_price)
  const output = parseNumber(model.output_price)
  const unitLabel = formatPriceUnit(model.price_unit)

  return {
    input: formatCnyAmount(input),
    output: formatCnyAmount(output),
    unitLabel,
  }
}

