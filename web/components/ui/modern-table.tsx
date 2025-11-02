"use client"

import * as React from "react"
import { motion, AnimatePresence } from "framer-motion"
import { Search, Filter, ChevronDown, ChevronUp, ArrowUpDown } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cn } from "@/lib/utils"

interface Column<T> {
  key: keyof T
  title: string
  render?: (value: any, record: T, index: number) => React.ReactNode
  sortable?: boolean
  width?: string
  className?: string
}

interface ModernTableProps<T> {
  data: T[]
  columns: Column<T>[]
  loading?: boolean
  searchable?: boolean
  searchPlaceholder?: string
  filterable?: boolean
  className?: string
  onRowClick?: (record: T, index: number) => void
  emptyText?: string
}

export function ModernTable<T extends Record<string, any>>({
  data,
  columns,
  loading = false,
  searchable = true,
  searchPlaceholder = "搜索...",
  filterable = true,
  className,
  onRowClick,
  emptyText = "暂无数据"
}: ModernTableProps<T>) {
  const [searchTerm, setSearchTerm] = React.useState("")
  const [sortConfig, setSortConfig] = React.useState<{
    key: keyof T | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })

  // 搜索过滤
  const filteredData = React.useMemo(() => {
    if (!searchTerm) return data
    
    return data.filter(item =>
      Object.values(item).some(value =>
        value?.toString().toLowerCase().includes(searchTerm.toLowerCase())
      )
    )
  }, [data, searchTerm])

  // 排序
  const sortedData = React.useMemo(() => {
    if (!sortConfig.key) return filteredData

    return [...filteredData].sort((a, b) => {
      const aValue = a[sortConfig.key!]
      const bValue = b[sortConfig.key!]

      if (aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1
      }
      if (aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1
      }
      return 0
    })
  }, [filteredData, sortConfig])

  const handleSort = (key: keyof T) => {
    setSortConfig(current => ({
      key,
      direction: current.key === key && current.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortIcon = (key: keyof T) => {
    if (sortConfig.key !== key) {
      return <ArrowUpDown className="h-4 w-4 opacity-50" />
    }
    
    return sortConfig.direction === 'asc' 
      ? <ChevronUp className="h-4 w-4" />
      : <ChevronDown className="h-4 w-4" />
  }

  if (loading) {
    return (
      <div className={cn("space-y-4", className)}>
        {/* Skeleton Header */}
        <div className="flex items-center justify-between">
          <div className="h-9 w-64 bg-muted rounded-lg animate-pulse" />
          <div className="h-9 w-20 bg-muted rounded-lg animate-pulse" />
        </div>
        
        {/* Skeleton Table */}
        <div className="border rounded-lg overflow-hidden">
          <div className="bg-muted/50 p-4 border-b">
            <div className="grid gap-4" style={{ gridTemplateColumns: columns.map(col => col.width || '1fr').join(' ') }}>
              {columns.map((_, index) => (
                <div key={index} className="h-4 bg-muted rounded animate-pulse" />
              ))}
            </div>
          </div>
          {[...Array(5)].map((_, index) => (
            <div key={index} className="p-4 border-b last:border-b-0">
              <div className="grid gap-4" style={{ gridTemplateColumns: columns.map(col => col.width || '1fr').join(' ') }}>
                {columns.map((_, colIndex) => (
                  <div key={colIndex} className="h-4 bg-muted rounded animate-pulse" />
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className={cn("space-y-4", className)}>
      {/* Search and Filter Bar */}
      {(searchable || filterable) && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          className="flex items-center justify-between gap-4"
        >
          {searchable && (
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="search"
                placeholder={searchPlaceholder}
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-9 input-modern"
              />
            </div>
          )}

          {filterable && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="btn-modern">
                  <Filter className="h-4 w-4 mr-2" />
                  筛选
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent className="glass-effect">
                <DropdownMenuItem>全部状态</DropdownMenuItem>
                <DropdownMenuItem>已激活</DropdownMenuItem>
                <DropdownMenuItem>已禁用</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </motion.div>
      )}

      {/* Modern Table */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="border rounded-lg overflow-hidden bg-card card-shadow"
      >
        {/* Table Header */}
        <div className="bg-muted/20 border-b">
          <div
            className="grid gap-4 p-4 font-medium text-sm"
            style={{ gridTemplateColumns: columns.map(col => col.width || '1fr').join(' ') }}
          >
            {columns.map((column) => (
              <div
                key={String(column.key)}
                className={cn(
                  "flex items-center gap-2",
                  column.sortable && "cursor-pointer hover:text-primary transition-colors select-none",
                  column.className
                )}
                onClick={() => column.sortable && handleSort(column.key)}
              >
                <span>{column.title}</span>
                {column.sortable && getSortIcon(column.key)}
              </div>
            ))}
          </div>
        </div>

        {/* Table Body */}
        <div className="divide-y">
          <AnimatePresence mode="wait">
            {sortedData.length === 0 ? (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className="p-12 text-center text-muted-foreground"
              >
                <div className="space-y-2">
                  <div className="text-lg font-medium">没有找到数据</div>
                  <div className="text-sm">{emptyText}</div>
                </div>
              </motion.div>
            ) : (
              sortedData.map((record, index) => (
                <motion.div
                  key={index}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.2, delay: index * 0.05 }}
                  className={cn(
                    "grid gap-4 p-4 hover:bg-muted/30 transition-colors",
                    onRowClick && "cursor-pointer",
                  )}
                  style={{ gridTemplateColumns: columns.map(col => col.width || '1fr').join(' ') }}
                  onClick={() => onRowClick?.(record, index)}
                >
                  {columns.map((column) => (
                    <div key={String(column.key)} className={cn("flex items-center", column.className)}>
                      {column.render
                        ? column.render(record[column.key], record, index)
                        : record[column.key]
                      }
                    </div>
                  ))}
                </motion.div>
              ))
            )}
          </AnimatePresence>
        </div>
      </motion.div>

      {/* Table Footer with Stats */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2 }}
        className="flex items-center justify-between text-sm text-muted-foreground"
      >
        <div>
          共 {data.length} 项，显示 {sortedData.length} 项
        </div>
        {searchTerm && (
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="text-xs">
              搜索: "{searchTerm}"
            </Badge>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSearchTerm("")}
              className="h-6 px-2 text-xs"
            >
              清除
            </Button>
          </div>
        )}
      </motion.div>
    </div>
  )
}
