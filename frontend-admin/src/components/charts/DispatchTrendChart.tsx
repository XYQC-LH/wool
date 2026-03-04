'use client';

import {
  ComposedChart,
  Line,
  Area,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

interface MetricPoint {
  time: string;
  request_count: number;
  success_rate: number;
  avg_latency_ms: number;
}

interface DispatchTrendChartProps {
  data: MetricPoint[];
}

export function DispatchTrendChart({ data }: DispatchTrendChartProps) {
  const formatTime = (time: string) => {
    const date = new Date(time);
    return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}`;
  };

  return (
    <ResponsiveContainer width="100%" height={300}>
      <ComposedChart data={data} margin={{ top: 20, right: 20, bottom: 20, left: 20 }}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis 
          dataKey="time" 
          tickFormatter={formatTime}
          tick={{ fontSize: 12 }}
        />
        <YAxis 
          yAxisId="left" 
          tick={{ fontSize: 12 }}
          label={{ value: '成功率 (%)', angle: -90, position: 'insideLeft' }}
        />
        <YAxis 
          yAxisId="right" 
          orientation="right" 
          tick={{ fontSize: 12 }}
          label={{ value: '延迟 (ms)', angle: 90, position: 'insideRight' }}
        />
        <Tooltip 
          contentStyle={{ backgroundColor: '#fff', border: '1px solid #ccc' }}
          labelFormatter={(label) => new Date(label).toLocaleString('zh-CN')}
        />
        <Legend />
        
        {/* 请求数 - 柱状图 */}
        <Bar 
          yAxisId="right"
          dataKey="request_count" 
          name="请求数" 
          fill="#f97316" 
          opacity={0.3}
          barSize={20}
        />
        
        {/* 成功率 - 面积图 */}
        <Area
          yAxisId="left"
          type="monotone"
          dataKey="success_rate"
          name="成功率 (%)"
          stroke="#22c55e"
          fill="#22c55e"
          fillOpacity={0.2}
          strokeWidth={2}
        />
        
        {/* 平均延迟 - 折线图 */}
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="avg_latency_ms"
          name="平均延迟 (ms)"
          stroke="#3b82f6"
          strokeWidth={2}
          dot={false}
        />
      </ComposedChart>
    </ResponsiveContainer>
  );
}
