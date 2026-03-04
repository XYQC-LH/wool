'use client';

import { useEffect, useState } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface TrafficPoint {
  time: string;
  requests: number;
}

interface RealtimeTrafficChartProps {
  refreshInterval?: number; // 刷新间隔（秒）
}

export function RealtimeTrafficChart({ refreshInterval = 5 }: RealtimeTrafficChartProps) {
  const [data, setData] = useState<TrafficPoint[]>([]);

  useEffect(() => {
    // 初始化数据（60个点，每5秒一个）
    const initialData: TrafficPoint[] = [];
    const now = new Date();
    for (let i = 60; i >= 0; i--) {
      const time = new Date(now.getTime() - i * refreshInterval * 1000);
      initialData.push({
        time: time.toISOString(),
        requests: Math.floor(Math.random() * 100) + 50,
      });
    }
    setData(initialData);

    // 定时更新
    const interval = setInterval(() => {
      setData(prev => {
        const newData = [...prev];
        const lastTime = new Date(newData[newData.length - 1].time);
        const newTime = new Date(lastTime.getTime() + refreshInterval * 1000);
        
        newData.push({
          time: newTime.toISOString(),
          requests: Math.floor(Math.random() * 100) + 50,
        });
        
        // 保持最多60个数据点
        if (newData.length > 60) {
          newData.shift();
        }
        
        return newData;
      });
    }, refreshInterval * 1000);

    return () => clearInterval(interval);
  }, [refreshInterval]);

  const formatTime = (time: string) => {
    const date = new Date(time);
    return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`;
  };

  return (
    <ResponsiveContainer width="100%" height={200}>
      <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.8}/>
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis 
          dataKey="time" 
          tickFormatter={formatTime}
          tick={{ fontSize: 10 }}
          interval="preserveStartEnd"
        />
        <YAxis 
          tick={{ fontSize: 10 }}
          domain={[0, 'auto']}
        />
        <Tooltip 
          contentStyle={{ backgroundColor: '#fff', border: '1px solid #ccc', fontSize: 12 }}
          labelFormatter={(label) => new Date(label).toLocaleTimeString('zh-CN')}
        />
        <Area
          type="monotone"
          dataKey="requests"
          stroke="#3b82f6"
          strokeWidth={2}
          fillOpacity={1}
          fill="url(#colorRequests)"
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}
