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

// 日间模式颜色
const LIGHT_COLORS = {
  primary: 'oklch(0.486 0.165 244.531)',
  foreground: 'oklch(0.15 0 0)',
  card: 'oklch(1 0 0)',
  cardForeground: 'oklch(0.15 0 0)',
  popover: 'oklch(1 0 0)',
  popoverForeground: 'oklch(0.15 0 0)',
  border: 'oklch(0.85 0 0)',
  mutedForeground: 'oklch(0.5 0 0)',
  chart: [
    'oklch(0.486 0.165 244.531)',
    'oklch(0.647 0.176 21.651)',
    'oklch(0.557 0.147 163.837)',
    'oklch(0.749 0.134 85.887)',
    'oklch(0.647 0.165 328.363)',
  ],
}

// 夜间模式颜色
const DARK_COLORS = {
  primary: 'oklch(0.628 0.185 244.531)',
  foreground: 'oklch(0.95 0 0)',
  card: 'oklch(0.12 0.008 240.146)',
  cardForeground: 'oklch(0.95 0 0)',
  popover: 'oklch(0.12 0.008 240.146)',
  popoverForeground: 'oklch(0.95 0 0)',
  border: 'oklch(0.3 0.015 240.146)',
  mutedForeground: 'oklch(0.7 0 0)',
  chart: [
    'oklch(0.628 0.185 244.531)',
    'oklch(0.747 0.196 21.651)',
    'oklch(0.667 0.167 163.837)',
    'oklch(0.829 0.154 85.887)',
    'oklch(0.767 0.185 328.363)',
  ],
}

interface ChartProps {
  data: any[]
  title?: string
  description?: string
  height?: number
  className?: string
}

export const ModernLineChart = memo(function ModernLineChart({ data, title, description, height = 300, className }: ChartProps) {
  const { theme, resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark' || theme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <LineChart data={data}>
        <defs>
          <linearGradient id="colorGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={colors.primary} stopOpacity={0.8}/>
            <stop offset="95%" stopColor={colors.primary} stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <CartesianGrid 
          strokeDasharray="3 3" 
          stroke={colors.border} 
          opacity={0.4}
        />
        <XAxis 
          dataKey="name" 
          stroke={colors.foreground}
          tick={{ fill: colors.foreground }}
          fontSize={12}
        />
        <YAxis 
          stroke={colors.foreground}
          tick={{ fill: colors.foreground }}
          fontSize={12}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: colors.popover,
            color: colors.popoverForeground,
            border: `1px solid ${colors.border}`,
            borderRadius: '8px',
            padding: '8px 12px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
          itemStyle={{
            color: colors.popoverForeground,
          }}
          labelStyle={{
            color: colors.popoverForeground,
          }}
          cursor={{ stroke: colors.border, strokeWidth: 1 }}
        />
        <Line
          type="monotone"
          dataKey="value"
          stroke={colors.primary}
          strokeWidth={2}
          dot={{ fill: colors.primary, strokeWidth: 2, r: 4 }}
          activeDot={{ r: 6, stroke: colors.primary, strokeWidth: 2 }}
        />
      </LineChart>
    </ResponsiveContainer>
  ), [data, height, colors])

  return (
    <div className={className}>
      <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
        {(title || description) && (
          <CardHeader>
            {title && <CardTitle>{title}</CardTitle>}
            {description && <CardDescription>{description}</CardDescription>}
          </CardHeader>
        )}
        <CardContent>
          {chartComponent}
        </CardContent>
      </Card>
    </div>
  )
})

export const ModernAreaChart = memo(function ModernAreaChart({ data, title, description, height = 300, className }: ChartProps) {
  const { theme, resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark' || theme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="colorUv" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={colors.primary} stopOpacity={0.6}/>
            <stop offset="95%" stopColor={colors.primary} stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <XAxis 
          dataKey="name" 
          stroke={colors.foreground}
          tick={{ fill: colors.foreground }}
          fontSize={12}
        />
        <YAxis 
          stroke={colors.foreground}
          tick={{ fill: colors.foreground }}
          fontSize={12}
        />
        <CartesianGrid 
          strokeDasharray="3 3" 
          stroke={colors.border} 
          opacity={0.4}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: colors.popover,
            color: colors.popoverForeground,
            border: `1px solid ${colors.border}`,
            borderRadius: '8px',
            padding: '8px 12px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
          itemStyle={{
            color: colors.popoverForeground,
          }}
          labelStyle={{
            color: colors.popoverForeground,
          }}
          cursor={{ stroke: colors.border, strokeWidth: 1 }}
        />
        <Area
          type="monotone"
          dataKey="value"
          stroke={colors.primary}
          fillOpacity={1}
          fill="url(#colorUv)"
        />
      </AreaChart>
    </ResponsiveContainer>
  ), [data, height, colors])

  return (
    <div className={className}>
      <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
        {(title || description) && (
          <CardHeader>
            {title && <CardTitle>{title}</CardTitle>}
            {description && <CardDescription>{description}</CardDescription>}
          </CardHeader>
        )}
        <CardContent>
          {chartComponent}
        </CardContent>
      </Card>
    </div>
  )
})

export const ModernBarChart = memo(function ModernBarChart({ data, title, description, height = 300, className }: ChartProps) {
  const { theme, resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark' || theme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <BarChart data={data}>
        <CartesianGrid 
          strokeDasharray="3 3" 
          stroke={colors.border} 
          opacity={0.4}
        />
        <XAxis 
          dataKey="name" 
          stroke={colors.foreground}
          tick={{ fill: colors.foreground }}
          fontSize={12}
        />
        <YAxis 
          stroke={colors.foreground}
          tick={{ fill: colors.foreground }}
          fontSize={12}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: colors.popover,
            color: colors.popoverForeground,
            border: `1px solid ${colors.border}`,
            borderRadius: '8px',
            padding: '8px 12px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
          itemStyle={{
            color: colors.popoverForeground,
          }}
          labelStyle={{
            color: colors.popoverForeground,
          }}
          cursor={{ stroke: colors.border, strokeWidth: 1 }}
        />
        <Bar 
          dataKey="value" 
          fill={colors.primary}
          radius={[4, 4, 0, 0]}
        />
      </BarChart>
    </ResponsiveContainer>
  ), [data, height, colors]);

  return (
    <div className={className}>
      <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
        {(title || description) && (
          <CardHeader>
            {title && <CardTitle>{title}</CardTitle>}
            {description && <CardDescription>{description}</CardDescription>}
          </CardHeader>
        )}
        <CardContent>
          {chartComponent}
        </CardContent>
      </Card>
    </div>
  )
})

export const ModernPieChart = memo(function ModernPieChart({ data, title, description, height = 300, className }: ChartProps) {
  const { theme, resolvedTheme } = useTheme()
  const isDark = resolvedTheme === 'dark' || theme === 'dark'
  const colors = isDark ? DARK_COLORS : LIGHT_COLORS

  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={60}
          outerRadius={100}
          paddingAngle={5}
          dataKey="value"
        >
          {data.map((entry, index) => (
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
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
          itemStyle={{
            color: colors.popoverForeground,
          }}
          labelStyle={{
            color: colors.popoverForeground,
          }}
          cursor={{ stroke: colors.border, strokeWidth: 1 }}
        />
        <Legend 
          wrapperStyle={{
            color: colors.cardForeground,
          }}
        />
      </PieChart>
    </ResponsiveContainer>
  ), [data, height, colors]);

  return (
    <div className={className}>
      <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
        {(title || description) && (
          <CardHeader>
            {title && <CardTitle>{title}</CardTitle>}
            {description && <CardDescription>{description}</CardDescription>}
          </CardHeader>
        )}
        <CardContent>
          {chartComponent}
        </CardContent>
      </Card>
    </div>
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
  const changeColor = useMemo(() => ({
    positive: 'text-green-600 dark:text-green-400',
    negative: 'text-red-600 dark:text-red-400',
    neutral: 'text-muted-foreground'
  }), [])

  return (
    <motion.div
      whileHover={{ scale: 1.02, y: -2 }}
      transition={{ type: "spring", stiffness: 400 }}
      className={className}
    >
      <Card className="card-shadow hover:card-shadow-lg transition-all duration-300 hover:border-primary/20">
        <CardContent className="p-6">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">{title}</p>
              <p className="text-3xl font-bold tracking-tight">{value}</p>
              {change && (
                <p className={`text-xs ${changeColor[changeType]} flex items-center gap-1`}>
                  {change}
                </p>
              )}
            </div>
            {icon && (
              <motion.div
                whileHover={{ scale: 1.1, rotate: 5 }}
                className="p-3 rounded-full bg-primary/10 text-primary"
              >
                {icon}
              </motion.div>
            )}
          </div>
        </CardContent>
      </Card>
    </motion.div>
  )
})
