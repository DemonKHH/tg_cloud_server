"use client"

import { memo, useMemo } from "react"
import { useTheme } from "next-themes"
import { 
  LineChart, 
  Line, 
  AreaChart, 
  Area, 
  BarChart, 
  Bar, 
  PieChart, 
  Pie, 
  Cell, 
  ResponsiveContainer, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  Legend 
} from "recharts"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { motion } from "framer-motion"
import { TrendingUp, TrendingDown, Minus } from "lucide-react"

// 日间模式颜色
const LIGHT_COLORS = {
  primary: '#4f46e5',
  foreground: '#1e1b4b',
  card: '#ffffff',
  cardForeground: '#1e1b4b',
  popover: '#ffffff',
  popoverForeground: '#1e1b4b',
  border: '#e5e7eb',
  mutedForeground: '#6b7280',
  chart: ['#4f46e5', '#10b981', '#f59e0b', '#ec4899', '#06b6d4'],
}

// 夜间模式颜色
const DARK_COLORS = {
  primary: '#818cf8',
  foreground: '#e0e7ff',
  card: '#1e1b4b',
  cardForeground: '#e0e7ff',
  popover: '#1e1b4b',
  popoverForeground: '#e0e7ff',
  border: '#3730a3',
  mutedForeground: '#a5b4fc',
  chart: ['#818cf8', '#34d399', '#fbbf24', '#f472b6', '#22d3ee'],
}

interface ChartProps {
  data: any[]
  title?: string
  description?: string
  height?: number
  className?: string
}

export const ModernLineChart = memo(function ModernLineChart({ data, title, description, height = 280, className }: ChartProps) {
  const { resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  return (
    <Card className={`border-border/50 ${className}`}>
      {(title || description) && (
        <CardHeader className="pb-2">
          {title && <CardTitle className="text-base font-semibold">{title}</CardTitle>}
          {description && <CardDescription className="text-xs">{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className="pt-0">
        <ResponsiveContainer width="100%" height={height}>
          <LineChart data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="lineGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={colors.primary} stopOpacity={0.3}/>
                <stop offset="95%" stopColor={colors.primary} stopOpacity={0}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke={colors.border} opacity={0.5} vertical={false} />
            <XAxis 
              dataKey="name" 
              stroke={colors.mutedForeground}
              tick={{ fill: colors.mutedForeground, fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <YAxis 
              stroke={colors.mutedForeground}
              tick={{ fill: colors.mutedForeground, fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: colors.popover,
                color: colors.popoverForeground,
                border: `1px solid ${colors.border}`,
                borderRadius: '8px',
                padding: '8px 12px',
                boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
                fontSize: '12px',
              }}
            />
            <Line
              type="monotone"
              dataKey="value"
              stroke={colors.primary}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4, fill: colors.primary, strokeWidth: 0 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
})

export const ModernAreaChart = memo(function ModernAreaChart({ data, title, description, height = 280, className }: ChartProps) {
  const { resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  return (
    <Card className={`border-border/50 ${className}`}>
      {(title || description) && (
        <CardHeader className="pb-2">
          {title && <CardTitle className="text-base font-semibold">{title}</CardTitle>}
          {description && <CardDescription className="text-xs">{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className="pt-0">
        <ResponsiveContainer width="100%" height={height}>
          <AreaChart data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={colors.primary} stopOpacity={0.4}/>
                <stop offset="95%" stopColor={colors.primary} stopOpacity={0.05}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke={colors.border} opacity={0.5} vertical={false} />
            <XAxis 
              dataKey="name" 
              stroke={colors.mutedForeground}
              tick={{ fill: colors.mutedForeground, fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <YAxis 
              stroke={colors.mutedForeground}
              tick={{ fill: colors.mutedForeground, fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: colors.popover,
                color: colors.popoverForeground,
                border: `1px solid ${colors.border}`,
                borderRadius: '8px',
                padding: '8px 12px',
                boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
                fontSize: '12px',
              }}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke={colors.primary}
              strokeWidth={2}
              fill="url(#areaGradient)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
})

export const ModernBarChart = memo(function ModernBarChart({ data, title, description, height = 280, className }: ChartProps) {
  const { resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  return (
    <Card className={`border-border/50 ${className}`}>
      {(title || description) && (
        <CardHeader className="pb-2">
          {title && <CardTitle className="text-base font-semibold">{title}</CardTitle>}
          {description && <CardDescription className="text-xs">{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className="pt-0">
        <ResponsiveContainer width="100%" height={height}>
          <BarChart data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke={colors.border} opacity={0.5} vertical={false} />
            <XAxis 
              dataKey="name" 
              stroke={colors.mutedForeground}
              tick={{ fill: colors.mutedForeground, fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <YAxis 
              stroke={colors.mutedForeground}
              tick={{ fill: colors.mutedForeground, fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: colors.popover,
                color: colors.popoverForeground,
                border: `1px solid ${colors.border}`,
                borderRadius: '8px',
                padding: '8px 12px',
                boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
                fontSize: '12px',
              }}
            />
            <Bar 
              dataKey="value" 
              fill={colors.primary}
              radius={[4, 4, 0, 0]}
              maxBarSize={40}
            />
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
})

export const ModernPieChart = memo(function ModernPieChart({ data, title, description, height = 280, className }: ChartProps) {
  const { resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  return (
    <Card className={`border-border/50 ${className}`}>
      {(title || description) && (
        <CardHeader className="pb-2">
          {title && <CardTitle className="text-base font-semibold">{title}</CardTitle>}
          {description && <CardDescription className="text-xs">{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className="pt-0">
        <ResponsiveContainer width="100%" height={height}>
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              innerRadius={55}
              outerRadius={85}
              paddingAngle={3}
              dataKey="value"
              strokeWidth={0}
            >
              {data.map((_, index) => (
                <Cell 
                  key={`cell-${index}`} 
                  fill={colors.chart[index % colors.chart.length]} 
                />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: colors.popover,
                color: colors.popoverForeground,
                border: `1px solid ${colors.border}`,
                borderRadius: '8px',
                padding: '8px 12px',
                boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
                fontSize: '12px',
              }}
            />
            <Legend 
              iconType="circle"
              iconSize={8}
              wrapperStyle={{ fontSize: '12px', color: colors.cardForeground }}
            />
          </PieChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
})

// 统计卡片组件
interface StatsCardProps {
  title: string
  value: string | number
  change?: string
  changeType?: 'positive' | 'negative' | 'neutral'
  icon?: React.ReactNode
  className?: string
}

export const StatsCard = memo(function StatsCard({ 
  title, 
  value, 
  change, 
  changeType = 'neutral', 
  icon, 
  className 
}: StatsCardProps) {
  const changeStyles = {
    positive: 'text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/50',
    negative: 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/50',
    neutral: 'text-muted-foreground bg-muted/50'
  }

  const ChangeIcon = changeType === 'positive' ? TrendingUp : changeType === 'negative' ? TrendingDown : Minus

  return (
    <Card className={`border-border/50 hover-card ${className}`}>
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">{title}</p>
            <p className="text-2xl font-bold tracking-tight">{value}</p>
            {change && (
              <div className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${changeStyles[changeType]}`}>
                <ChangeIcon className="h-3 w-3" />
                {change}
              </div>
            )}
          </div>
          {icon && (
            <div className="p-2.5 rounded-xl bg-primary/10 text-primary">
              {icon}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
})
