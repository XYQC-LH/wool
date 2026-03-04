'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';

interface CostItem {
  provider_name: string;
  channel_name: string;
  cost: number;
  revenue: number;
  profit: number;
}

interface CostAnalysisChartProps {
  data: CostItem[];
}

export function CostAnalysisChart({ data }: CostAnalysisChartProps) {
  // 按成本排序并取前10个
  const sortedData = [...data]
    .sort((a, b) => b.cost - a.cost)
    .slice(0, 10)
    .map(item => ({
      ...item,
      name: `${item.provider_name}(${item.channel_name})`,
    }));

  return (
    <ResponsiveContainer width="100%" height={350}>
      <BarChart data={sortedData} layout="vertical" margin={{ top: 20, right: 30, left: 100, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" horizontal={false} />
        <XAxis type="number" tick={{ fontSize: 12 }} />
        <YAxis 
          type="category" 
          dataKey="name" 
          tick={{ fontSize: 11 }}
          width={90}
        />
        <Tooltip 
          formatter={(value: number) => `$${value.toFixed(4)}`}
          contentStyle={{ backgroundColor: '#fff', border: '1px solid #ccc' }}
        />
        <Legend />
        
        {/* 成本 */}
        <Bar dataKey="cost" name="成本" fill="#ef4444" barSize={15} />
        
        {/* 收入 */}
        <Bar dataKey="revenue" name="收入" fill="#3b82f6" barSize={15} />
        
        {/* 利润 */}
        <Bar dataKey="profit" name="利润" fill="#22c55e" barSize={15} />
      </BarChart>
    </ResponsiveContainer>
  );
}
