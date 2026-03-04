'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Loading } from '@/components/common/loading';
import { useToast } from '@/components/ui/use-toast';
import api, { getErrorMessage } from '@/lib/api';
import {
  RefreshCw,
  Activity,
  CheckCircle,
  Clock,
  Server,
  TrendingUp,
  TrendingDown,
} from 'lucide-react';

const debugWarn = (...args: unknown[]) => {
  if (process.env.NODE_ENV !== 'production') {
    console.warn(...args);
  }
};

// 类型定义
interface ProviderStatItem {
  id: number;
  name: string;
  channel_name: string;
  cost_per_1k: number;
  status: string;
  circuit_state: string;
  success_rate: number;
  avg_latency_ms: number;
  request_count: number;
  traffic_percentage: number;
  health_score: number;
}

interface DispatchStats {
  total_requests: number;
  success_rate: number;
  avg_latency_ms: number;
  active_providers: number;
  circuit_open_providers: number;
  total_providers: number;
  providers: ProviderStatItem[];
}

interface MetricPoint {
  time: string;
  request_count: number;
  success_rate: number;
  avg_latency_ms: number;
}

interface CircuitEvent {
  id: number;
  provider_id: number;
  provider_name: string;
  channel_name: string;
  event_type: string;
  reason: string;
  created_at: string;
}

interface Model {
  id: string;
  name: string;
}

// 统计卡片组件
function StatCard({
  title,
  value,
  change,
  icon: Icon,
  trend,
}: {
  title: string;
  value: string | number;
  change?: string;
  icon: React.ElementType;
  trend?: 'up' | 'down' | 'neutral';
}) {
  return (
    <Card className="p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-500">{title}</p>
          <p className="text-2xl font-bold mt-1">{value}</p>
          {change && (
            <div className="flex items-center mt-1">
              {trend === 'up' && <TrendingUp className="w-4 h-4 text-green-500 mr-1" />}
              {trend === 'down' && <TrendingDown className="w-4 h-4 text-red-500 mr-1" />}
              <span className={`text-sm ${
                trend === 'up' ? 'text-green-500' : trend === 'down' ? 'text-red-500' : 'text-gray-500'
              }`}>
                {change}
              </span>
            </div>
          )}
        </div>
        <div className="p-3 bg-orange-100 rounded-full">
          <Icon className="w-6 h-6 text-orange-600" />
        </div>
      </div>
    </Card>
  );
}

// 状态标签组件
function StatusBadge({ status, circuitState }: { status: string; circuitState: string }) {
  if (circuitState === 'open') {
    return (
      <span className="px-2 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
        🔴 熔断
      </span>
    );
  }
  if (circuitState === 'half_open') {
    return (
      <span className="px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
        🟡 半开
      </span>
    );
  }
  if (status === 'active') {
    return (
      <span className="px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
        🟢 健康
      </span>
    );
  }
  return (
    <span className="px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
      ⚪ 禁用
    </span>
  );
}

// 流量分布条组件
function TrafficBar({ providers }: { providers: ProviderStatItem[] }) {
  const colors = [
    'bg-blue-500',
    'bg-green-500',
    'bg-yellow-500',
    'bg-purple-500',
    'bg-pink-500',
    'bg-indigo-500',
  ];

  return (
    <div className="space-y-2">
      <div className="flex h-8 rounded-lg overflow-hidden">
        {providers.map((p, i) => (
          <div
            key={p.id}
            className={`${colors[i % colors.length]} transition-all`}
            style={{ width: `${p.traffic_percentage}%` }}
            title={`${p.name}: ${Number(p.traffic_percentage).toFixed(1)}%`}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-4">
        {providers.map((p, i) => (
          <div key={p.id} className="flex items-center gap-2">
            <div className={`w-3 h-3 rounded ${colors[i % colors.length]}`} />
            <span className="text-sm">
              {p.channel_name} ({Number(p.traffic_percentage).toFixed(1)}%)
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// 简单折线图组件
function SimpleLineChart({ data, dataKey, color }: { data: MetricPoint[]; dataKey: keyof MetricPoint; color: string }) {
  if (!data || data.length === 0) {
    return <div className="h-32 flex items-center justify-center text-gray-400">暂无数据</div>;
  }

  const values = data.map(d => Number(d[dataKey]));
  const max = Math.max(...values);
  const min = Math.min(...values);
  const range = max - min || 1;

  const points = data.map((d, i) => {
    const x = (i / (data.length - 1)) * 100;
    const y = 100 - ((Number(d[dataKey]) - min) / range) * 80 - 10;
    return `${x},${y}`;
  }).join(' ');

  return (
    <div className="h-32 relative">
      <svg className="w-full h-full" viewBox="0 0 100 100" preserveAspectRatio="none">
        <polyline
          fill="none"
          stroke={color}
          strokeWidth="2"
          points={points}
        />
      </svg>
      <div className="absolute top-0 right-0 text-xs text-gray-500">{Number(max).toFixed(1)}</div>
      <div className="absolute bottom-0 right-0 text-xs text-gray-500">{Number(min).toFixed(1)}</div>
    </div>
  );
}

// 熔断事件时间线组件
function CircuitEventTimeline({ events }: { events: CircuitEvent[] }) {
  if (!events || events.length === 0) {
    return <div className="text-center text-gray-400 py-4">暂无熔断事件</div>;
  }

  return (
    <div className="space-y-3">
      {events.slice(0, 10).map((event) => (
        <div key={event.id} className="flex items-start gap-3">
          <div className={`mt-1 w-2 h-2 rounded-full ${
            event.event_type === 'open' ? 'bg-red-500' : 'bg-green-500'
          }`} />
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium">{event.provider_name}</span>
              <span className="text-sm text-gray-500">({event.channel_name})</span>
              <span className={`text-xs px-2 py-0.5 rounded ${
                event.event_type === 'open' ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'
              }`}>
                {event.event_type === 'open' ? '触发熔断' : '熔断恢复'}
              </span>
            </div>
            <p className="text-sm text-gray-600 mt-1">{event.reason || '无详细原因'}</p>
            <p className="text-xs text-gray-400 mt-1">
              {new Date(event.created_at).toLocaleString('zh-CN')}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}

export default function DispatchPage() {
  const { toast } = useToast();
  const lastErrorToastAtRef = useRef(0);
  const reportError = useCallback(
    (title: string, error: unknown) => {
      const now = Date.now();
      const message = getErrorMessage(error);

      if (now - lastErrorToastAtRef.current < 15_000) {
        debugWarn(title, message);
        return;
      }

      lastErrorToastAtRef.current = now;
      toast({ title, description: message, variant: 'destructive' });
      debugWarn(title, message);
    },
    [toast]
  );
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);  // ⭐ 新增：自动刷新开关
  const [refreshInterval] = useState(30);  // ⭐ 新增：刷新间隔（秒）
  const [stats, setStats] = useState<DispatchStats | null>(null);
  const [metrics, setMetrics] = useState<MetricPoint[]>([]);
  const [events, setEvents] = useState<CircuitEvent[]>([]);
  const [models, setModels] = useState<Model[]>([]);
  const [selectedModel, setSelectedModel] = useState('all');
  const [period, setPeriod] = useState('24h');

  // 加载统计数据
  const loadStats = useCallback(async () => {
    try {
      const params: Record<string, string> = { period };
      if (selectedModel && selectedModel !== 'all') {
        params.model_id = selectedModel;
      }

      const response = await api.get('/api/admin/dispatch/stats', { params });
      
      // 更安全的类型检查
      if (!response || typeof response !== 'object') {
        debugWarn('loadStats: 无效的响应格式', response);
        setStats({
          total_requests: 0,
          success_rate: 0,
          avg_latency_ms: 0,
          active_providers: 0,
          circuit_open_providers: 0,
          total_providers: 0,
          providers: [],
        });
        return;
      }

      // 检查 success 字段
      if ('success' in response && response.success === false) {
        debugWarn('loadStats: API 返回失败', response);
        setStats({
          total_requests: 0,
          success_rate: 0,
          avg_latency_ms: 0,
          active_providers: 0,
          circuit_open_providers: 0,
          total_providers: 0,
          providers: [],
        });
        return;
      }

      // 安全地提取数据
      if ('data' in response && response.data && typeof response.data === 'object') {
        setStats(response.data as DispatchStats);
      } else {
        debugWarn('loadStats: 响应格式异常', response);
        setStats({
          total_requests: 0,
          success_rate: 0,
          avg_latency_ms: 0,
          active_providers: 0,
          circuit_open_providers: 0,
          total_providers: 0,
          providers: [],
        });
      }
    } catch (error) {
      reportError('加载统计数据失败', error);
      setStats({
        total_requests: 0,
        success_rate: 0,
        avg_latency_ms: 0,
        active_providers: 0,
        circuit_open_providers: 0,
        total_providers: 0,
        providers: [],
      });
    }
  }, [selectedModel, period, reportError]);

  // 加载指标数据
  const loadMetrics = useCallback(async () => {
    try {
      const params: Record<string, string> = { period, granularity: 'hour' };
      if (selectedModel && selectedModel !== 'all') {
        params.model_id = selectedModel;
      }

      const response = await api.get('/api/admin/dispatch/metrics', { params });
      
      // 更安全的类型检查
      if (!response || typeof response !== 'object') {
        debugWarn('loadMetrics: 无效的响应格式', response);
        setMetrics([]);
        return;
      }

      // 检查 success 字段
      if ('success' in response && response.success === false) {
        debugWarn('loadMetrics: API 返回失败', response);
        setMetrics([]);
        return;
      }

      // 安全地提取 metrics 数据
      let metricsData: MetricPoint[] = [];
      if ('data' in response && response.data && typeof response.data === 'object') {
        if ('metrics' in response.data && Array.isArray(response.data.metrics)) {
          metricsData = response.data.metrics;
        }
      }

      setMetrics(metricsData);
    } catch (error) {
      reportError('加载指标数据失败', error);
      setMetrics([]);
    }
  }, [selectedModel, period, reportError]);

  // 加载熔断事件
  const loadEvents = useCallback(async () => {
    try {
      const response = await api.get('/api/admin/dispatch/events', { params: { page_size: '10' } });
      
      // 更安全的类型检查
      if (!response || typeof response !== 'object') {
        debugWarn('loadEvents: 无效的响应格式', response);
        setEvents([]);
        return;
      }

      // 检查 success 字段
      if ('success' in response && response.success === false) {
        debugWarn('loadEvents: API 返回失败', response);
        setEvents([]);
        return;
      }

      // 安全地提取事件数据
      if ('data' in response && response.data && typeof response.data === 'object') {
        if ('list' in response.data && Array.isArray(response.data.list)) {
          setEvents(response.data.list);
        } else {
          setEvents([]);
        }
      } else {
        debugWarn('loadEvents: 响应格式异常', response);
        setEvents([]);
      }
    } catch (error) {
      reportError('加载熔断事件失败', error);
      setEvents([]);
    }
  }, [reportError]);

  // 加载模型列表
  const loadModels = useCallback(async () => {
    try {
      const response = await api.get('/api/admin/models', { params: { page_size: '100' } });
      const data = response as { data?: { list?: Model[] } };
      if (data.data?.list) {
        setModels(data.data.list);
      }
    } catch (error) {
      reportError('加载模型失败', error);
    }
  }, [reportError]);

  // 刷新所有数据
  const refreshAll = async () => {
    setRefreshing(true);
    await Promise.all([loadStats(), loadMetrics(), loadEvents()]);
    setRefreshing(false);
    toast({ title: '数据已刷新' });
  };

  // 初始加载
  useEffect(() => {
    const init = async () => {
      setLoading(true);
      await loadModels();
      await Promise.all([loadStats(), loadMetrics(), loadEvents()]);
      setLoading(false);
    };
    init();
  }, [loadEvents, loadMetrics, loadModels, loadStats]);

  // ⭐ 新增：自动刷新机制
  useEffect(() => {
    if (!autoRefresh) {
      return;  // 如果自动刷新关闭，不启动定时器
    }

    const intervalId = setInterval(async () => {
      try {
        await Promise.all([loadStats(), loadMetrics(), loadEvents()]);
      } catch (error) {
        reportError('自动刷新失败', error);
      }
    }, refreshInterval * 1000);  // 转换为毫秒

    return () => clearInterval(intervalId);  // 清理定时器
  }, [autoRefresh, refreshInterval, loadStats, loadMetrics, loadEvents, reportError]);

  // 筛选条件变化时重新加载
  useEffect(() => {
    if (!loading) {
      loadStats();
      loadMetrics();
    }
  }, [selectedModel, period, loadStats, loadMetrics, loading]);

  if (loading) {
    return <Loading />;
  }

  return (
    <div className="space-y-6">
      {/* 页面标题 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold">智能调度系统</h1>
          <p className="text-gray-500 mt-1">实时监控调度状态、源头健康度和熔断事件</p>
        </div>
        <div className="flex items-center gap-4">
          <Select value={selectedModel} onValueChange={setSelectedModel}>
            <SelectTrigger className="w-40">
              <SelectValue placeholder="全部模型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部模型</SelectItem>
              {models.map((model) => (
                <SelectItem key={model.id} value={model.id}>
                  {model.name || model.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={period} onValueChange={setPeriod}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">最近1小时</SelectItem>
              <SelectItem value="6h">最近6小时</SelectItem>
              <SelectItem value="24h">最近24小时</SelectItem>
              <SelectItem value="7d">最近7天</SelectItem>
            </SelectContent>
          </Select>
          {/* ⭐ 新增：自动刷新控制 */}
          <div className="flex items-center gap-2 border rounded-lg px-3 py-2">
            <input
              type="checkbox"
              id="autoRefresh"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="w-4 h-4"
            />
            <label htmlFor="autoRefresh" className="text-sm cursor-pointer">
              自动刷新
            </label>
            {autoRefresh && (
              <span className="text-xs text-gray-500">
                ({refreshInterval}秒)
              </span>
            )}
          </div>
          <Button onClick={refreshAll} disabled={refreshing}>
            <RefreshCw className={`w-4 h-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="总请求数"
          value={stats?.total_requests?.toLocaleString() || '0'}
          icon={Activity}
        />
        <StatCard
          title="成功率"
          value={`${(stats?.success_rate ?? 0).toFixed(1)}%`}
          icon={CheckCircle}
          trend={stats?.success_rate !== undefined && stats.success_rate >= 99 ? 'up' : 'down'}
        />
        <StatCard
          title="平均延迟"
          value={`${stats?.avg_latency_ms ?? 0}ms`}
          icon={Clock}
        />
        <StatCard
          title="活跃源头"
          value={`${stats?.active_providers ?? 0}/${stats?.total_providers ?? 0}`}
          change={stats?.circuit_open_providers !== undefined && stats.circuit_open_providers > 0 ? `${stats.circuit_open_providers}个熔断中` : undefined}
          icon={Server}
          trend={stats?.circuit_open_providers !== undefined && stats.circuit_open_providers > 0 ? 'down' : 'neutral'}
        />
      </div>

      {/* 流量分布 */}
      <Card className="p-6">
        <h2 className="text-lg font-semibold mb-4">调度流量分布</h2>
        {stats?.providers && stats.providers.length > 0 ? (
          <TrafficBar providers={stats.providers} />
        ) : (
          <div className="text-center text-gray-400 py-4">暂无流量数据</div>
        )}
      </Card>

      {/* 源头列表 */}
      <Card className="p-6">
        <h2 className="text-lg font-semibold mb-4">源头健康状态</h2>
        {stats?.providers && stats.providers.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">排序</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">源头名称</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">渠道</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">成本/1K</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">状态</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">成功率</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">延迟</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">请求数</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">健康分</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {stats.providers
                  .sort((a, b) => a.cost_per_1k - b.cost_per_1k)
                  .map((provider, index) => (
                    <tr key={provider.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-sm">{index + 1}</td>
                      <td className="px-4 py-3 font-medium">{provider.name}</td>
                      <td className="px-4 py-3 text-sm text-gray-500">{provider.channel_name}</td>
                      <td className="px-4 py-3 text-sm">${(provider.cost_per_1k ?? 0).toFixed(4)}</td>
                      <td className="px-4 py-3">
                        <StatusBadge status={provider.status} circuitState={provider.circuit_state} />
                      </td>
                      <td className="px-4 py-3 text-sm">
                        {provider.circuit_state === 'open' ? '-' : `${(provider.success_rate ?? 0).toFixed(1)}%`}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        {provider.circuit_state === 'open' ? '-' : `${provider.avg_latency_ms}ms`}
                      </td>
                      <td className="px-4 py-3 text-sm">{provider.request_count.toLocaleString()}</td>
                      <td className="px-4 py-3">
                        <span className={`font-medium ${
                          (provider.health_score ?? 0) >= 80 ? 'text-green-600' :
                          (provider.health_score ?? 0) >= 50 ? 'text-yellow-600' : 'text-red-600'
                        }`}>
                          {(provider.health_score ?? 0).toFixed(1)}
                        </span>
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center text-gray-400 py-4">暂无源头数据</div>
        )}
      </Card>

      {/* 趋势图表和熔断事件 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 趋势图表 */}
        <Card className="p-6">
          <h2 className="text-lg font-semibold mb-4">调度趋势</h2>
          <div className="space-y-6">
            <div>
              <p className="text-sm text-gray-500 mb-2">成功率 (%)</p>
              <SimpleLineChart data={metrics} dataKey="success_rate" color="#22c55e" />
            </div>
            <div>
              <p className="text-sm text-gray-500 mb-2">平均延迟 (ms)</p>
              <SimpleLineChart data={metrics} dataKey="avg_latency_ms" color="#3b82f6" />
            </div>
            <div>
              <p className="text-sm text-gray-500 mb-2">请求数</p>
              <SimpleLineChart data={metrics} dataKey="request_count" color="#f97316" />
            </div>
          </div>
        </Card>

        {/* 熔断事件 */}
        <Card className="p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">熔断事件记录</h2>
            <Button variant="outline" size="sm" onClick={() => window.location.href = '/dashboard/dispatch/events'}>
              查看全部
            </Button>
          </div>
          <CircuitEventTimeline events={events} />
        </Card>
      </div>

      {/* 状态说明 */}
      <Card className="p-4">
        <div className="flex flex-wrap gap-6 text-sm">
          <div className="flex items-center gap-2">
            <span className="px-2 py-1 rounded-full bg-green-100 text-green-800">🟢 健康</span>
            <span className="text-gray-500">成功率 &gt; 98%, 延迟正常</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="px-2 py-1 rounded-full bg-yellow-100 text-yellow-800">🟡 半开</span>
            <span className="text-gray-500">熔断恢复中，探测请求</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="px-2 py-1 rounded-full bg-red-100 text-red-800">🔴 熔断</span>
            <span className="text-gray-500">连续失败达阈值，暂停服务</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="px-2 py-1 rounded-full bg-gray-100 text-gray-800">⚪ 禁用</span>
            <span className="text-gray-500">管理员手动禁用</span>
          </div>
        </div>
      </Card>
    </div>
  );
}
