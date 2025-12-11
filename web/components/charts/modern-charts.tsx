"use client"

import { memo, ReactNode } from "react"
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
  Legend,
  RadialBarChart,
  RadialBar,
  ComposedChart,
  TooltipProps,
} from "recharts"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Minus } from "lucide-react"
import { cn } from "@/lib/utils"

// 颜色配置
const LIGHT_COLORS = {
  primary: '#4f46e5',
  secondary: '#8b5cf6',
  foreground: '#1e1b4b',
  card: '#ffffff',
  cardForeground: '#1e1b4b',
  popover: '#ffffff',
  popoverForeground: '#1e1b4b',
  border: '#e5e7eb',
  mutedForeground: '#6b7280',
  grid: '#f3f4f6',
  chart: ['#4f46e5', '#10b981', '#f59e0b', '#ec4899', '#06b6d4', '#8b5cf6', '#f97316'],
}

const DARK_COLORS = {
  primary: '#818cf8',
  secondary: '#a78bfa',
  foreground: '#e0e7ff',
  card: '#1e1b4b',
  cardForeground: '#e0e7ff',
  popover: '#1f2937',
  popoverForeground: '#e0e7ff',
  border: '#374151',
  mutedForeground: '#9ca3af',
  grid: '#374151',
  chart: ['#818cf8', '#34d399', '#fbbf24', '#f472b6', '#22d3ee', '#a78bfa', '#fb923c'],
}

// 使用主题颜色的 Hook
function useChartColors() {
  const { resolvedTheme } = useTheme()
  return resolvedTheme === 'dark' ? DARK_COLORS : LIGHT_COLORS
}

// 自定义 Tooltip 组件
interface CustomTooltipProps {
  active?: boolean
  payload?: Array<{
    value: number | string
    name?: string
    color?: string
    dataKey?: string
  }>
  label?: string
  colors: typeof LIGHT_COLORS
}

const CustomTooltip = ({ active, payload, label, colors }: CustomTooltipProps) => {
  if (!active || !payload?.length) return null

  return (
    <div
      className="rounded-lg border shadow-lg p-3 min-w-[120px]"
      style={{
        backgroundColor: colors.popover,
        borderColor: colors.border,
      }}
    >
      <p className="text-xs font-medium mb-2" style={{ color: colors.mutedForeground }}>
        {label}
      </p>
      {payload.map((entry, index: number) => (
        <div key={index} className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <div
              className="w-2 h-2 rounded-full"
              style={{ backgroundColor: entry.color }}
            />
            <span className="text-xs" style={{ color: colors.popoverForeground }}>
              {entry.name || 'Value'}
            </span>
          </div>
          <span className="text-xs font-semibold" style={{ color: colors.popoverForeground }}>
            {typeof entry.value === 'number' ? entry.value.toLocaleString() : entry.value}
          </span>
        </div>
      ))}
    </div>
  )
}

// 基础图表 Props
interface ChartProps {
  data: any[]
  title?: string
  description?: string
  height?: number
  className?: string
  dataKey?: string
  nameKey?: string
}

// 多数据系列 Props
interface MultiSeriesChartProps extends ChartProps {
  series?: { dataKey: string; name: string; color?: string }[]
}

// 折线图
export const ModernLineChart = memo(function ModernLineChart({
  data,
  title,
  description,
  height = 280,
  className,
  dataKey = "value",
  nameKey = "name",
}: ChartProps) {
  const colors = useChartColors()

  if (!data?.length) {
    return (
      <ChartCard title={title} description={description} className={className}>
        <EmptyChart height={height} />
      </ChartCard>
    )
  }

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
          <defs>
            <linearGradient id="lineGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={colors.primary} stopOpacity={0.2} />
              <stop offset="95%" stopColor={colors.primary} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={colors.grid} vertical={false} />
          <XAxis
            dataKey={nameKey}
            stroke={colors.mutedForeground}
            tick={{ fill: colors.mutedForeground, fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            dy={10}
          />
          <YAxis
            stroke={colors.mutedForeground}
            tick={{ fill: colors.mutedForeground, fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            dx={-10}
          />
          <Tooltip content={<CustomTooltip colors={colors} />} />
          <Line
            type="monotone"
            dataKey={dataKey}
            stroke={colors.primary}
            strokeWidth={2.5}
            dot={false}
            activeDot={{ r: 5, fill: colors.primary, strokeWidth: 2, stroke: colors.card }}
          />
        </LineChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 面积图
export const ModernAreaChart = memo(function ModernAreaChart({
  data,
  title,
  description,
  height = 280,
  className,
  dataKey = "value",
  nameKey = "name",
}: ChartProps) {
  const colors = useChartColors()

  if (!data?.length) {
    return (
      <ChartCard title={title} description={description} className={className}>
        <EmptyChart height={height} />
      </ChartCard>
    )
  }

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <AreaChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
          <defs>
            <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={colors.primary} stopOpacity={0.3} />
              <stop offset="95%" stopColor={colors.primary} stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={colors.grid} vertical={false} />
          <XAxis
            dataKey={nameKey}
            stroke={colors.mutedForeground}
            tick={{ fill: colors.mutedForeground, fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            dy={10}
          />
          <YAxis
            stroke={colors.mutedForeground}
            tick={{ fill: colors.mutedForeground, fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            dx={-10}
          />
          <Tooltip content={<CustomTooltip colors={colors} />} />
          <Area
            type="monotone"
            dataKey={dataKey}
            stroke={colors.primary}
            strokeWidth={2}
            fill="url(#areaGradient)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 柱状图
export const ModernBarChart = memo(function ModernBarChart({
  data,
  title,
  description,
  height = 280,
  className,
  dataKey = "value",
  nameKey = "name",
}: ChartProps) {
  const colors = useChartColors()

  if (!data?.length) {
    return (
      <ChartCard title={title} description={description} className={className}>
        <EmptyChart height={height} />
      </ChartCard>
    )
  }

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={colors.grid} vertical={false} />
          <XAxis
            dataKey={nameKey}
            stroke={colors.mutedForeground}
            tick={{ fill: colors.mutedForeground, fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            dy={10}
          />
          <YAxis
            stroke={colors.mutedForeground}
            tick={{ fill: colors.mutedForeground, fontSize: 11 }}
            axisLine={false}
            tickLine={false}
            dx={-10}
          />
          <Tooltip content={<CustomTooltip colors={colors} />} />
          <Bar
            dataKey={dataKey}
            fill={colors.primary}
            radius={[4, 4, 0, 0]}
            maxBarSize={45}
          />
        </BarChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 饼图
export const ModernPieChart = memo(function ModernPieChart({
  data,
  title,
  description,
  height = 280,
  className,
}: ChartProps) {
  const colors = useChartColors()

  if (!data?.length) {
    return (
      <ChartCard title={title} description={description} className={className}>
        <EmptyChart height={height} />
      </ChartCard>
    )
  }

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={80}
            paddingAngle={2}
            dataKey="value"
            strokeWidth={0}
          >
            {data.map((_, index) => (
              <Cell key={`cell-${index}`} fill={colors.chart[index % colors.chart.length]} />
            ))}
          </Pie>
          <Tooltip content={<CustomTooltip colors={colors} />} />
          <Legend
            iconType="circle"
            iconSize={8}
            formatter={(value) => (
              <span style={{ color: colors.cardForeground, fontSize: '12px' }}>{value}</span>
            )}
          />
        </PieChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 环形进度图
interface RadialProgressProps {
  value: number
  maxValue?: number
  title?: string
  description?: string
  label?: string
  height?: number
  className?: string
}

export const RadialProgress = memo(function RadialProgress({
  value,
  maxValue = 100,
  title,
  description,
  label,
  height = 200,
  className,
}: RadialProgressProps) {
  const colors = useChartColors()
  const percentage = Math.round((value / maxValue) * 100)
  const data = [{ name: label || 'Progress', value: percentage, fill: colors.primary }]

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <RadialBarChart
          cx="50%"
          cy="50%"
          innerRadius="60%"
          outerRadius="90%"
          barSize={12}
          data={data}
          startAngle={90}
          endAngle={-270}
        >
          <RadialBar
            background={{ fill: colors.grid }}
            dataKey="value"
            cornerRadius={6}
          />
          <text
            x="50%"
            y="50%"
            textAnchor="middle"
            dominantBaseline="middle"
            className="text-2xl font-bold"
            fill={colors.foreground}
          >
            {percentage}%
          </text>
        </RadialBarChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 多系列折线图
export const MultiLineChart = memo(function MultiLineChart({
  data,
  title,
  description,
  height = 280,
  className,
  series = [],
  nameKey = "name",
}: MultiSeriesChartProps) {
  const colors = useChartColors()

  if (!data?.length) {
    return (
      <ChartCard title={title} description={description} className={className}>
        <EmptyChart height={height} />
      </ChartCard>
    )
  }

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={colors.grid} vertical={false} />
          <XAxis
            dataKey={nameKey}
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
          <Tooltip content={<CustomTooltip colors={colors} />} />
          <Legend
            iconType="circle"
            iconSize={8}
            formatter={(value) => (
              <span style={{ color: colors.cardForeground, fontSize: '12px' }}>{value}</span>
            )}
          />
          {series.map((s, index) => (
            <Line
              key={s.dataKey}
              type="monotone"
              dataKey={s.dataKey}
              name={s.name}
              stroke={s.color || colors.chart[index % colors.chart.length]}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 组合图（柱状 + 折线）
export const ComboChart = memo(function ComboChart({
  data,
  title,
  description,
  height = 280,
  className,
  barDataKey = "value",
  lineDataKey = "trend",
  nameKey = "name",
}: ChartProps & { barDataKey?: string; lineDataKey?: string }) {
  const colors = useChartColors()

  if (!data?.length) {
    return (
      <ChartCard title={title} description={description} className={className}>
        <EmptyChart height={height} />
      </ChartCard>
    )
  }

  return (
    <ChartCard title={title} description={description} className={className}>
      <ResponsiveContainer width="100%" height={height}>
        <ComposedChart data={data} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={colors.grid} vertical={false} />
          <XAxis
            dataKey={nameKey}
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
          <Tooltip content={<CustomTooltip colors={colors} />} />
          <Bar dataKey={barDataKey} fill={colors.primary} radius={[4, 4, 0, 0]} maxBarSize={40} />
          <Line
            type="monotone"
            dataKey={lineDataKey}
            stroke={colors.chart[1]}
            strokeWidth={2}
            dot={false}
          />
        </ComposedChart>
      </ResponsiveContainer>
    </ChartCard>
  )
})

// 图表卡片容器
interface ChartCardProps {
  title?: string
  description?: string
  className?: string
  children: ReactNode
}

function ChartCard({ title, description, className, children }: ChartCardProps) {
  return (
    <Card className={cn("border-border/50", className)}>
      {(title || description) && (
        <CardHeader className="pb-2">
          {title && <CardTitle className="text-base font-semibold">{title}</CardTitle>}
          {description && <CardDescription className="text-xs">{description}</CardDescription>}
        </CardHeader>
      )}
      <CardContent className="pt-0">{children}</CardContent>
    </Card>
  )
}

// 空图表占位
function EmptyChart({ height }: { height: number }) {
  return (
    <div
      className="flex items-center justify-center text-muted-foreground text-sm"
      style={{ height }}
    >
      暂无数据
    </div>
  )
}

// 统计卡片组件
interface StatsCardProps {
  title: string
  value: string | number
  change?: string
  changeType?: 'positive' | 'negative' | 'neutral'
  icon?: ReactNode
  className?: string
}

export const StatsCard = memo(function StatsCard({
  title,
  value,
  change,
  changeType = 'neutral',
  icon,
  className,
}: StatsCardProps) {
  const changeStyles = {
    positive: 'text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/50',
    negative: 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/50',
    neutral: 'text-muted-foreground bg-muted/50',
  }

  const ChangeIcon =
    changeType === 'positive' ? TrendingUp : changeType === 'negative' ? TrendingDown : Minus

  return (
    <Card className={cn("border-border/50 hover-card", className)}>
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">{title}</p>
            <p className="text-2xl font-bold tracking-tight">{value}</p>
            {change && (
              <div
                className={cn(
                  "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium",
                  changeStyles[changeType]
                )}
              >
                <ChangeIcon className="h-3 w-3" />
                {change}
              </div>
            )}
          </div>
          {icon && <div className="p-2.5 rounded-xl bg-primary/10 text-primary">{icon}</div>}
        </div>
      </CardContent>
    </Card>
  )
})

// 迷你图表（用于表格或卡片内）
interface SparklineProps {
  data: number[]
  width?: number
  height?: number
  color?: string
  className?: string
}

export const Sparkline = memo(function Sparkline({
  data,
  width = 100,
  height = 30,
  color,
  className,
}: SparklineProps) {
  const colors = useChartColors()
  const chartData = data.map((value, index) => ({ index, value }))

  return (
    <div className={className} style={{ width, height }}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} margin={{ top: 2, right: 2, left: 2, bottom: 2 }}>
          <defs>
            <linearGradient id="sparklineGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color || colors.primary} stopOpacity={0.3} />
              <stop offset="95%" stopColor={color || colors.primary} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="value"
            stroke={color || colors.primary}
            strokeWidth={1.5}
            fill="url(#sparklineGradient)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
})
