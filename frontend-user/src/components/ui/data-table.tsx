'use client'

import { useState } from 'react'
import { ChevronUp, ChevronDown, Filter, X } from 'lucide-react'
import { Button } from './button'
import { Input } from './input'

export interface Column<T> {
  key: keyof T
  label: string
  sortable?: boolean
  filterable?: boolean
  render?: (value: unknown, row: T) => React.ReactNode
  width?: string
}

export interface DataTableProps<T> {
  data: T[]
  columns: Column<T>[]
  loading?: boolean
  emptyMessage?: string
  onRowClick?: (row: T) => void
}

export function DataTable<T extends Record<string, unknown>>({
  data,
  columns,
  loading = false,
  emptyMessage = '暂无数据',
  onRowClick,
}: DataTableProps<T>) {
  const [sortColumn, setSortColumn] = useState<keyof T | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [showFilters, setShowFilters] = useState(false)

  // 排序数据
  const sortedData = [...data].sort((a, b) => {
    if (!sortColumn) return 0
    
    const aValue = a[sortColumn]
    const bValue = b[sortColumn]
    
    if (aValue === bValue) return 0
    if (aValue === null || aValue === undefined) return 1
    if (bValue === null || bValue === undefined) return -1
    
    let comparison = 0
    if (typeof aValue === 'string' && typeof bValue === 'string') {
      comparison = aValue.localeCompare(bValue)
    } else if (typeof aValue === 'number' && typeof bValue === 'number') {
      comparison = aValue - bValue
    } else {
      comparison = String(aValue).localeCompare(String(bValue))
    }
    
    return sortDirection === 'asc' ? comparison : -comparison
  })

  // 过滤数据
  const filteredData = sortedData.filter((row) => {
    return Object.entries(filters).every(([key, value]) => {
      if (!value) return true
      const cellValue = row[key]
      if (cellValue === null || cellValue === undefined) return false
      return String(cellValue).toLowerCase().includes(value.toLowerCase())
    })
  })

  // 处理排序
  const handleSort = (column: Column<T>) => {
    if (!column.sortable) return
    
    if (sortColumn === column.key) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortColumn(column.key)
      setSortDirection('asc')
    }
  }

  // 处理过滤
  const handleFilter = (column: Column<T>, value: string) => {
    setFilters(prev => ({
      ...prev,
      [String(column.key)]: value,
    }))
  }

  // 清除过滤
  const clearFilter = (column: Column<T>) => {
    setFilters(prev => {
      const newFilters = { ...prev }
      delete newFilters[String(column.key)]
      return newFilters
    })
  }

  // 清除所有过滤
  const clearAllFilters = () => {
    setFilters({})
  }

  // 获取排序图标
  const getSortIcon = (column: Column<T>) => {
    if (sortColumn !== column.key) return null
    return sortDirection === 'asc' ? (
      <ChevronUp className="h-4 w-4" />
    ) : (
      <ChevronDown className="h-4 w-4" />
    )
  }

  return (
    <div className="space-y-4">
      {/* 过滤器工具栏 */}
      {columns.some(col => col.filterable) && (
        <div className="flex items-center justify-between">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowFilters(!showFilters)}
          >
            <Filter className="mr-2 h-4 w-4" />
            筛选
            {Object.keys(filters).length > 0 && (
              <span className="ml-2 px-2 py-0.5 bg-primary text-primary-foreground text-xs rounded-full">
                {Object.keys(filters).length}
              </span>
            )}
          </Button>
          {Object.keys(filters).length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={clearAllFilters}
            >
              <X className="mr-2 h-4 w-4" />
              清除筛选
            </Button>
          )}
        </div>
      )}

      {/* 过滤器面板 */}
      {showFilters && columns.some(col => col.filterable) && (
        <div className="p-4 border rounded-lg bg-muted/50 space-y-3">
          {columns
            .filter(col => col.filterable)
            .map((column) => (
              <div key={String(column.key)} className="space-y-2">
                <label className="text-sm font-medium">{column.label}</label>
                <div className="flex gap-2">
                  <Input
                    placeholder={`搜索${column.label}...`}
                    value={filters[String(column.key)] || ''}
                    onChange={(e) => handleFilter(column, e.target.value)}
                    className="flex-1"
                  />
                  {filters[String(column.key)] && (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => clearFilter(column)}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>
            ))}
        </div>
      )}

      {/* 数据表格 */}
      <div className="overflow-x-auto border rounded-lg">
        <table className="w-full">
          <thead className="bg-muted">
            <tr>
              {columns.map((column) => (
                <th
                  key={String(column.key)}
                  className={`px-4 py-3 text-left text-sm font-medium text-muted-foreground ${
                    column.sortable ? 'cursor-pointer hover:bg-muted/80 transition-colors' : ''
                  }`}
                  style={{ width: column.width }}
                  onClick={() => handleSort(column)}
                >
                  <div className="flex items-center gap-2">
                    {column.label}
                    {column.sortable && getSortIcon(column)}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-8 text-center">
                  <div className="space-y-2">
                    {[1, 2, 3].map((i) => (
                      <div key={i} className="h-12 bg-muted rounded animate-pulse" />
                    ))}
                  </div>
                </td>
              </tr>
            ) : filteredData.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-12 text-center">
                  <p className="text-muted-foreground">{emptyMessage}</p>
                </td>
              </tr>
            ) : (
              filteredData.map((row, index) => (
                <tr
                  key={index}
                  className={`border-b hover:bg-accent/50 transition-colors ${
                    onRowClick ? 'cursor-pointer' : ''
                  }`}
                  onClick={() => onRowClick?.(row)}
                >
                  {columns.map((column) => (
                    <td key={String(column.key)} className="px-4 py-3 text-sm">
                      {column.render
                        ? column.render(row[column.key], row)
                        : String(row[column.key] ?? '')}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* 数据统计 */}
      {!loading && (
        <div className="text-sm text-muted-foreground">
          显示 {filteredData.length} 条记录，共 {data.length} 条
        </div>
      )}
    </div>
  )
}
