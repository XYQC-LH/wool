'use client';

import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts';

interface TrafficItem {
  name: string;
  value: number;
  cost: number;
}

interface TrafficPieChartProps {
  data: TrafficItem[];
}

const COLORS = [
  '#22c55e', // 绿色 - 低成本
  '#3b82f6', // 蓝色
  '#f59e0b', // 黄色
  '#ef4444', // 红色 - 高成本
  '#8b5cf6', // 紫色
  '#ec4899', // 粉色
];

export function TrafficPieChart({ data }: TrafficPieChartProps) {
  const sortedData = [...data].sort((a, b) => a.cost - b.cost);
  
  return (
    <ResponsiveContainer width="100%" height={300}>
      <PieChart>
        <Pie
          data={sortedData}
          cx="50%"
          cy="50%"
          innerRadius={60}
          outerRadius={100}
          paddingAngle={5}
          dataKey="value"
          nameKey="name"
          label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
        >
          {sortedData.map((entry, index) => (
            <Cell 
              key={`cell-${index}`} 
              fill={COLORS[index % COLORS.length]}
              stroke="#fff"
              strokeWidth={2}
            />
          ))}
        </Pie>
        <Tooltip 
          formatter={(value: number, name: string, props: any) => {
            const item = props.payload;
            return [
              `${value.toLocaleString()} (${item.percent}%)`,
              `${item.name} (成本: $${item.cost.toFixed(4)}/1K)`
            ];
          }}
        />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  );
}
