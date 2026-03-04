'use client';

import {
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';

interface LatencyPoint {
  time: string;
  p50: number;
  p95: number;
  p99: number;
  request_count: number;
}

interface LatencyPercentileChartProps {
  data: LatencyPoint[];
}

export function LatencyPercentileChart({ data }: LatencyPercentileChartProps) {
  const formatTime = (time: string) => {
    const date = new Date(time);
    return `${date.getHours()}:00`;
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
          tick={{ fontSize: 12 }}
          label={{ value: '延迟 (ms)', angle: -90, position: 'insideLeft' }}
        />
        <Tooltip 
          contentStyle={{ backgroundColor: '#fff', border: '1px solid #ccc' }}
          labelFormatter={(label) => new Date(label).toLocaleString('zh-CN')}
          formatter={(value: number, name: string) => {
            return [`${value}ms`, name];
          }}
        />
        <Legend />
        
        {/* P50 - 中位数 */}
        <Line
          type="monotone"
          dataKey="p50"
          name="P50 (中位数)"
          stroke="#22c55e"
          strokeWidth={2}
          dot={{ r: 3 }}
        />
        
        {/* P95 */}
        <Line
          type="monotone"
          dataKey="p95"
          name="P95"
          stroke="#f59e0b"
          strokeWidth={2}
          dot={{ r: 3 }}
        />
        
        {/* P99 */}
        <Line
          type="monotone"
          dataKey="p99"
          name="P99"
          stroke="#ef4444"
          strokeWidth={2}
          dot={{ r: 3 }}
        />

        {/* 参考线 - 预期延迟 */}
        <ReferenceLine y={200} stroke="#94a3b8" strokeDasharray="5 5" label="预期延迟 200ms" />
      </ComposedChart>
    </ResponsiveContainer>
  );
}
