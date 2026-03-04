import PayClient from '../pay-client'
import { CreditCard } from 'lucide-react'

export default async function StripePayPage({ searchParams }: { searchParams?: Promise<{ order_no?: string | string[] }> }) {
  const resolved = await searchParams
  const raw = resolved?.order_no
  const orderNo = Array.isArray(raw) ? raw[0] : raw
  return (
    <PayClient
      title="信用卡支付"
      methodLabel="信用卡（Stripe）"
      icon={<CreditCard className="w-5 h-5" />}
      orderNo={orderNo || ''}
    />
  )
}
