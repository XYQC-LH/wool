'use client';

interface HeatmapPoint {
  hour: number;
  day: string;
  open_count: number;
  intensity: number;
}

interface CircuitHeatmapProps {
  data: HeatmapPoint[];
}

export function CircuitHeatmap({ data }: CircuitHeatmapProps) {
  // 提取唯一的天数
  const days = [...new Set(data.map(d => d.day))].reverse();
  const hours = Array.from({ length: 24 }, (_, i) => i);

  // 获取强度对应的颜色
  const getColor = (intensity: number) => {
    if (intensity === 0) return '#f3f4f6'; // gray-100
    if (intensity < 0.3) return '#fee2e2'; // red-100
    if (intensity < 0.6) return '#fecaca'; // red-200
    if (intensity < 0.8) return '#f87171'; // red-400
    return '#dc2626'; // red-600
  };

  // 查找特定时间点的数据
  const getPoint = (day: string, hour: number) => {
    return data.find(d => d.day === day && d.hour === hour) || {
      hour,
      day,
      open_count: 0,
      intensity: 0
    };
  };

  return (
    <div className="overflow-x-auto">
      <div className="min-w-[600px]">
        {/* 表头 - 小时 */}
        <div className="flex">
          <div className="w-16 flex-shrink-0"></div>
          {hours.map(hour => (
            <div 
              key={hour} 
              className="w-8 flex-shrink-0 text-xs text-center text-gray-500"
            >
              {hour}
            </div>
          ))}
        </div>

        {/* 表格内容 */}
        {days.map(day => (
          <div key={day} className="flex items-center mt-1">
            <div className="w-16 flex-shrink-0 text-xs text-gray-600 font-medium">
              {day}
            </div>
            {hours.map(hour => {
              const point = getPoint(day, hour);
              return (
                <div
                  key={`${day}-${hour}`}
                  className="w-8 h-8 flex-shrink-0 rounded-sm mx-0.5 transition-colors hover:ring-2 hover:ring-offset-1 hover:ring-blue-400"
                  style={{ backgroundColor: getColor(point.intensity) }}
                  title={`${day} ${hour}:00 - 熔断 ${point.open_count} 次`}
                >
                  {point.open_count > 0 && (
                    <span className="flex items-center justify-center h-full text-xs text-gray-700 font-medium">
                      {point.open_count}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        ))}

        {/* 图例 */}
        <div className="flex items-center gap-4 mt-4 text-xs text-gray-500">
          <span>熔断次数:</span>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded" style={{ backgroundColor: '#f3f4f6' }}></div>
            <span>0</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded" style={{ backgroundColor: '#fee2e2' }}></div>
            <span>1-2</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded" style={{ backgroundColor: '#fecaca' }}></div>
            <span>3-5</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded" style={{ backgroundColor: '#f87171' }}></div>
            <span>6-8</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-4 h-4 rounded" style={{ backgroundColor: '#dc2626' }}></div>
            <span>9+</span>
          </div>
        </div>
      </div>
    </div>
  );
}
