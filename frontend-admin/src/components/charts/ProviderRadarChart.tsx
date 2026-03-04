'use client';

import {
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  Radar,
  ResponsiveContainer,
  Tooltip,
  Legend,
} from 'recharts';

interface RadarDataPoint {
  metric: string;
  provider1: number;
  provider2?: number;
  provider3?: number;
}

interface ProviderRadarChartProps {
  providers: Array<{
    id: number;
    name: string;
    success_rate: number;
    avg_latency_ms: number;
    cost_per_1k: number;
    health_score: number;
    traffic_percent: number;
  }>;
  selectedProviders: number[];
}

export function ProviderRadarChart({ providers, selectedProviders }: ProviderRadarChartProps) {
  // 标准化数据到0-100范围
  const maxLatency = Math.max(...providers.map(p => p.avg_latency_ms), 1);
  const maxCost = Math.max(...providers.map(p => p.cost_per_1k), 0.01);

  const data: RadarDataPoint[] = [
    {
      metric: '成功率',
      ...providers.reduce((acc, p) => {
        if (selectedProviders.includes(p.id)) {
          acc[`provider${p.id}`] = p.success_rate;
        }
        return acc;
      }, {} as any)
    },
    {
      metric: '响应速度',
      ...providers.reduce((acc, p) => {
        if (selectedProviders.includes(p.id)) {
          // 延迟越低成本越低，反向计算
          acc[`provider${p.id}`] = Math.max(0, 100 - (p.avg_latency_ms / maxLatency) * 100);
        }
        return acc;
      }, {} as any)
    },
    {
      metric: '成本效益',
      ...providers.reduce((acc, p) => {
        if (selectedProviders.includes(p.id)) {
          // 成本越低分数越高
          acc[`provider${p.id}`] = Math.max(0, 100 - (p.cost_per_1k / maxCost) * 100);
        }
        return acc;
      }, {} as any)
    },
    {
      metric: '健康度',
      ...providers.reduce((acc, p) => {
        if (selectedProviders.includes(p.id)) {
          acc[`provider${p.id}`] = p.health_score;
        }
        return acc;
      }, {} as any)
    },
    {
      metric: '流量占比',
      ...providers.reduce((acc, p) => {
        if (selectedProviders.includes(p.id)) {
          acc[`provider${p.id}`] = Math.min(p.traffic_percent * 5, 100); // 放大比例以便观察
        }
        return acc;
      }, {} as any)
    },
  ];

  const colors = ['#3b82f6', '#22c55e', '#f59e0b', '#ef4444', '#8b5cf6'];

  return (
    <ResponsiveContainer width="100%" height={300}>
      <RadarChart cx="50%" cy="50%" outerRadius="80%" data={data}>
        <PolarGrid />
        <PolarAngleAxis dataKey="metric" tick={{ fontSize: 12 }} />
        <PolarRadiusAxis angle={30} domain={[0, 100]} tick={false} axisLine={false} />
        
        {selectedProviders.map((providerId, index) => {
          const provider = providers.find(p => p.id === providerId);
          return (
            <Radar
              key={providerId}
              name={provider?.name || `源头${providerId}`}
              dataKey={`provider${providerId}`}
              stroke={colors[index % colors.length]}
              fill={colors[index % colors.length]}
              fillOpacity={0.2}
              strokeWidth={2}
            />
          );
        })}
        
        <Tooltip />
        <Legend />
      </RadarChart>
    </ResponsiveContainer>
  );
}
