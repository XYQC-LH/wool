import PayClient from '../pay-client'
import { Wallet } from 'lucide-react'

export default async function CryptoPayPage({ searchParams }: { searchParams?: Promise<{ order_no?: string | string[] }> }) {
  const resolved = await searchParams
  const raw = resolved?.order_no
  const orderNo = Array.isArray(raw) ? raw[0] : raw
  return <PayClient title="加密货币支付" methodLabel="加密货币" icon={<Wallet className="w-5 h-5" />} orderNo={orderNo || ''} />
}
