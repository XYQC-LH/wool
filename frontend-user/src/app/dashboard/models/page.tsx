'use client'

import { useEffect, useMemo, useState } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { SkeletonCard } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/use-toast'
import { publicApi, type Model } from '@/lib/api'
import { copyToClipboard } from '@/lib/utils'
import {
  getModelCategory,
  getModelCategoryLabel,
  formatModelPricing,
  type ModelCategory,
  type ModelCategoryFilter,
} from '@/lib/model-utils'
import {
  Cpu,
  Search,
  Copy,
  ExternalLink,
  Image as ImageIcon,
  Video as VideoIcon,
  Music,
  MessageSquareText,
  CheckCircle2,
  XCircle,
} from 'lucide-react'

type SortKey = 'name_asc' | 'price_asc' | 'context_desc'

function ProviderAvatar({ provider }: { provider: string }) {
  const initials = (provider || '?')
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0]?.toUpperCase())
    .join('')

  return (
    <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center text-sm font-semibold text-muted-foreground flex-shrink-0">
      {initials || '?'}
    </div>
  )
}

function CategoryIcon({ category }: { category: ModelCategoryFilter }) {
  if (category === 'image') return <ImageIcon className="w-4 h-4" />
  if (category === 'video') return <VideoIcon className="w-4 h-4" />
  if (category === 'audio') return <Music className="w-4 h-4" />
  return <MessageSquareText className="w-4 h-4" />
}

function StatusPill({ enabled }: { enabled: boolean }) {
  return (
    <div
      className={[
        'inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs font-medium',
        enabled ? 'bg-green-500/10 text-green-600' : 'bg-muted text-muted-foreground',
      ].join(' ')}
    >
      {enabled ? <CheckCircle2 className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
      {enabled ? '可用' : '不可用'}
    </div>
  )
}

function Chip({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">
      {children}
    </span>
  )
}

export default function ModelsPage() {
  const [loading, setLoading] = useState(true)
  const [models, setModels] = useState<Model[]>([])

  const [category, setCategory] = useState<ModelCategoryFilter>('all')
  const [onlyEnabled, setOnlyEnabled] = useState(true)
  const [providerFilter, setProviderFilter] = useState<string[] | null>(null) // null = 全部
  const [providerQuery, setProviderQuery] = useState('')
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('name_asc')

  useEffect(() => {
    const load = async () => {
      setLoading(true)
      try {
        const res = await publicApi.getModels()
        if (res.code === 0) {
          setModels(res.data || [])
        } else {
          setModels([])
          toast({
            title: '加载失败',
            description: res.message || '无法获取公开模型列表，请稍后重试',
            variant: 'destructive',
          })
        }
      } catch {
        toast({
          title: '加载失败',
          description: '无法获取公开模型列表，请稍后重试',
          variant: 'destructive',
        })
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  const providers = useMemo(() => {
    const all = Array.from(
      new Set(models.map((m) => (m.provider || '').trim()).filter(Boolean))
    )
    all.sort((a, b) => a.localeCompare(b))
    return all
  }, [models])

  const hasDisabledModels = useMemo(() => models.some((m) => !m.enabled), [models])

  const visibleProviders = useMemo(() => {
    const q = providerQuery.trim().toLowerCase()
    if (!q) return providers
    return providers.filter((p) => p.toLowerCase().includes(q))
  }, [providerQuery, providers])

  const baseFilteredModels = useMemo(() => {
    const query = search.trim().toLowerCase()

    return models.filter((m) => {
      if (onlyEnabled && !m.enabled) return false

      if (category !== 'all' && getModelCategory(m) !== category) return false

      if (!query) return true
      const haystack = `${m.display_name} ${m.name} ${m.provider} ${m.description || ''}`.toLowerCase()
      return haystack.includes(query)
    })
  }, [models, category, onlyEnabled, search])

  const providerSummaries = useMemo(() => {
    const categoryOrder: ModelCategory[] = ['text', 'image', 'video', 'audio']

    const map = new Map<string, { provider: string; count: number; categories: Set<ModelCategory> }>()
    for (const m of baseFilteredModels) {
      const provider = (m.provider || '').trim()
      if (!provider) continue

      const entry = map.get(provider) ?? { provider, count: 0, categories: new Set<ModelCategory>() }
      entry.count += 1
      entry.categories.add(getModelCategory(m))
      map.set(provider, entry)
    }

    const list = Array.from(map.values()).map((entry) => ({
      provider: entry.provider,
      count: entry.count,
      categories: categoryOrder.filter((c) => entry.categories.has(c)),
    }))

    list.sort((a, b) => b.count - a.count || a.provider.localeCompare(b.provider))
    return list
  }, [baseFilteredModels])

  const filteredModels = useMemo(() => {
    const query = search.trim().toLowerCase()

    const activeProviders = providerFilter === null ? null : new Set(providerFilter)

    const list = models.filter((m) => {
      if (onlyEnabled && !m.enabled) return false

      if (category !== 'all' && getModelCategory(m) !== category) return false

      if (activeProviders && !activeProviders.has(m.provider)) return false

      if (!query) return true
      const haystack = `${m.display_name} ${m.name} ${m.provider} ${m.description || ''}`.toLowerCase()
      return haystack.includes(query)
    })

    const getContext = (m: Model) => Math.max(m.context_length || 0, m.max_context || 0)
    const getPrice = (m: Model) => {
      const price = Number(String(m.input_price))
      if (!Number.isFinite(price)) return Number.POSITIVE_INFINITY

      const unit = Number(m.price_unit)
      if (!Number.isFinite(unit) || unit <= 0) return price

      return price * (1000 / unit)
    }

    list.sort((a, b) => {
      if (sortKey === 'context_desc') return getContext(b) - getContext(a)
      if (sortKey === 'price_asc') return getPrice(a) - getPrice(b)
      return a.name.localeCompare(b.name)
    })

    return list
  }, [models, category, onlyEnabled, providerFilter, search, sortKey])

  const focusProvider = (provider: string) => {
    setProviderFilter((prev) => {
      if (prev?.length === 1 && prev[0] === provider) return null
      return [provider]
    })
  }

  const toggleProvider = (provider: string, checked: boolean) => {
    setProviderFilter((prev) => {
      const current = prev ?? providers
      const next = new Set(current)
      if (checked) next.add(provider)
      else next.delete(provider)

      const arr = Array.from(next)
      arr.sort((a, b) => a.localeCompare(b))
      if (arr.length === providers.length) return null
      return arr
    })
  }

  const onCopy = async (text: string) => {
    await copyToClipboard(text)
    toast({ title: '已复制', description: text })
  }

  const hasActiveFilters = category !== 'all' || !!search.trim() || providerFilter !== null || (hasDisabledModels && onlyEnabled)

  const resetFilters = () => {
    setCategory('all')
    setOnlyEnabled(true)
    setProviderFilter(null)
    setProviderQuery('')
    setSearch('')
    setSortKey('name_asc')
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Cpu className="w-7 h-7" />
          模型列表
        </h1>
        <p className="text-muted-foreground mt-1">
          浏览公开可用的模型，支持按模态分类、筛选与快速检索（未登录也可查看）。
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-4">
        <div className="lg:col-span-1">
          <Card className="lg:sticky lg:top-20">
            <CardHeader>
              <CardTitle className="text-lg">分类与筛选</CardTitle>
              <CardDescription>快速定位你需要的模型</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="space-y-2">
                <div className="text-sm font-medium">模态分类</div>
                <div className="space-y-1">
                  {(['all', 'text', 'image', 'video', 'audio'] as const).map((key) => (
                    <button
                      key={key}
                      onClick={() => setCategory(key)}
                      className={[
                        'w-full flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition-colors',
                        category === key
                          ? 'bg-primary text-primary-foreground'
                          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                      ].join(' ')}
                    >
                      <CategoryIcon category={key} />
                      <span>{getModelCategoryLabel(key)}</span>
                      {category === key && <span className="ml-auto text-xs opacity-80">当前</span>}
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-2">
                <div className="text-sm font-medium">状态</div>
                <label className="flex items-center gap-2 text-sm text-muted-foreground select-none">
                  <input
                    type="checkbox"
                    className="rounded"
                    checked={onlyEnabled}
                    onChange={(e) => setOnlyEnabled(e.target.checked)}
                  />
                  仅看可用
                </label>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="text-sm font-medium">提供商</div>
                  <div className="flex items-center gap-2">
                    {providers.length > 0 && (
                      <div className="text-xs text-muted-foreground whitespace-nowrap">
                        {(providerFilter === null ? providers.length : providerFilter.length)}/{providers.length}
                      </div>
                    )}
                    <Button size="sm" variant="ghost" className="h-7 px-2" onClick={() => setProviderFilter(null)}>
                      全选
                    </Button>
                    <Button size="sm" variant="ghost" className="h-7 px-2" onClick={() => setProviderFilter([])}>
                      清空
                    </Button>
                  </div>
                </div>

                {providers.length === 0 ? (
                  <div className="text-sm text-muted-foreground">暂无数据</div>
                ) : (
                  <>
                    <Input
                      placeholder="搜索提供商..."
                      value={providerQuery}
                      onChange={(e) => setProviderQuery(e.target.value)}
                      className="h-9"
                    />

                    {visibleProviders.length === 0 ? (
                      <div className="text-sm text-muted-foreground">暂无匹配的提供商</div>
                    ) : (
                      <div className="space-y-2 max-h-56 overflow-auto pr-1">
                        {visibleProviders.map((p) => {
                          const checked = providerFilter === null ? true : providerFilter.includes(p)
                          return (
                            <label
                              key={p}
                              className="flex items-center gap-2 text-sm text-muted-foreground select-none"
                            >
                              <input
                                type="checkbox"
                                className="rounded"
                                checked={checked}
                                onChange={(e) => toggleProvider(p, e.target.checked)}
                              />
                              <span className="truncate">{p}</span>
                            </label>
                          )
                        })}
                      </div>
                    )}
                  </>
                )}
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="lg:col-span-3 space-y-4">
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <CardTitle className="text-lg">模型厂商</CardTitle>
                  <CardDescription>按厂商快速定位模型列表</CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  {!loading && (
                    <div className="text-sm text-muted-foreground whitespace-nowrap">
                      {providerSummaries.length} 个
                    </div>
                  )}
                  {providerFilter !== null && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-8"
                      onClick={() => setProviderFilter(null)}
                    >
                      清除厂商筛选
                    </Button>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="text-sm text-muted-foreground">加载中...</div>
              ) : providerSummaries.length === 0 ? (
                <div className="text-sm text-muted-foreground">暂无厂商数据</div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
                  {providerSummaries.map((summary) => {
                    const selected =
                      providerFilter !== null && providerFilter.length > 0 && providerFilter.includes(summary.provider)

                    return (
                      <button
                        key={summary.provider}
                        type="button"
                        onClick={() => focusProvider(summary.provider)}
                        className={[
                          'w-full text-left rounded-xl border bg-card p-4 transition-colors',
                          'hover:bg-accent hover:text-accent-foreground',
                          selected ? 'ring-2 ring-primary' : '',
                        ].join(' ')}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex items-start gap-3 min-w-0">
                            <ProviderAvatar provider={summary.provider} />
                            <div className="min-w-0">
                              <div className="font-medium truncate">{summary.provider}</div>
                              <div className="text-sm text-muted-foreground">{summary.count} 个模型</div>
                            </div>
                          </div>
                          {selected && (
                            <div className="text-xs text-primary font-medium whitespace-nowrap">已选</div>
                          )}
                        </div>

                        {summary.categories.length > 0 && (
                          <div className="mt-3 flex flex-wrap gap-2">
                            {summary.categories.map((c) => (
                              <span
                                key={c}
                                className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground"
                              >
                                <CategoryIcon category={c} />
                                {getModelCategoryLabel(c)}
                              </span>
                            ))}
                          </div>
                        )}
                      </button>
                    )
                  })}
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardContent className="pt-6">
              <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <Input
                    placeholder="搜索模型 / 提供商 / 描述..."
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="pl-10"
                  />
                </div>

                <div className="flex items-center gap-2">
                  <div className="text-sm text-muted-foreground whitespace-nowrap">
                    共 <span className="text-foreground font-medium">{filteredModels.length}</span> 个
                  </div>
                  <select
                    value={sortKey}
                    onChange={(e) => setSortKey(e.target.value as SortKey)}
                    className="h-10 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  >
                    <option value="name_asc">按名称</option>
                    <option value="price_asc">按输入价格(低→高)</option>
                    <option value="context_desc">按上下文(高→低)</option>
                  </select>
                </div>
              </div>

              {hasActiveFilters && (
                <div className="flex flex-wrap items-center gap-2 pt-4">
                  {category !== 'all' && <Chip>分类：{getModelCategoryLabel(category)}</Chip>}
                  {!!search.trim() && <Chip>搜索：{search.trim()}</Chip>}
                  {providerFilter !== null && <Chip>厂商：{providerFilter.length} 个</Chip>}
                  {hasDisabledModels && onlyEnabled && <Chip>仅看可用</Chip>}

                  <Button size="sm" variant="ghost" className="h-7 px-2" onClick={resetFilters}>
                    重置筛选
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>

          {loading ? (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {Array.from({ length: 6 }).map((_, i) => (
                <SkeletonCard key={i} />
              ))}
            </div>
          ) : filteredModels.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                暂无符合条件的模型
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {filteredModels.map((model) => {
                const modelCategory = getModelCategory(model)
                const pricing = formatModelPricing(model)
                const context = Math.max(model.context_length || 0, model.max_context || 0)
                const title = model.display_name || model.name

                return (
                  <Card key={model.id} className="group hover:shadow-md transition-shadow">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex items-start gap-3 min-w-0">
                          <ProviderAvatar provider={model.provider} />
                          <div className="min-w-0">
                            <CardTitle className="text-lg leading-tight truncate">{title}</CardTitle>
                            <CardDescription className="truncate">
                              {model.provider} · <span className="font-mono">{model.name}</span>
                            </CardDescription>
                          </div>
                        </div>
                        <div className="flex flex-col items-end gap-2 flex-shrink-0">
                          <StatusPill enabled={model.enabled} />
                          <Button
                            size="icon"
                            variant="ghost"
                            className="h-8 w-8"
                            onClick={() => onCopy(model.name)}
                            aria-label="复制模型名称"
                          >
                            <Copy className="w-4 h-4" />
                          </Button>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <p className="text-sm text-muted-foreground line-clamp-2 min-h-[40px]">
                        {model.description || '暂无描述'}
                      </p>

                      <div className="flex flex-wrap gap-2">
                        <Chip>
                          <span className="inline-flex items-center gap-1">
                            <CategoryIcon category={modelCategory} />
                            {getModelCategoryLabel(modelCategory)}
                          </span>
                        </Chip>
                        {context > 0 && <Chip>上下文 {context.toLocaleString()}</Chip>}
                        <Chip>
                          输入 {pricing.input} / {pricing.unitLabel}
                        </Chip>
                        <Chip>
                          输出 {pricing.output} / {pricing.unitLabel}
                        </Chip>
                      </div>

                      <div className="flex items-center justify-end">
                        <Link href={`/dashboard/models/${encodeURIComponent(model.id)}`}>
                          <Button variant="outline" className="gap-2">
                            查看详情
                            <ExternalLink className="w-4 h-4" />
                          </Button>
                        </Link>
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
