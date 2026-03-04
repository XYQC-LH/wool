import ModelDetailClient from './model-detail-client'

export default async function ModelDetailPage({ params }: { params?: Promise<{ id?: string | string[] }> }) {
  const resolved = await params
  const rawId = resolved?.id
  const id = Array.isArray(rawId) ? rawId[0] : rawId
  return <ModelDetailClient id={id || ''} />
}

