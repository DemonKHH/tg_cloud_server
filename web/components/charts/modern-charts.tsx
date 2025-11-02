"use client"

import { memo, useMemo } from "react"
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

const COLORS = [
  'oklch(var(--chart-1))',
  'oklch(var(--chart-2))',
  'oklch(var(--chart-3))',
  'oklch(var(--chart-4))',
  'oklch(var(--chart-5))',
]

interface ChartProps {
  data: any[]
  title?: string
  description?: string
  height?: number
  className?: string
}

export const ModernLineChart = memo(function ModernLineChart({ data, title, description, height = 300, className }: ChartProps) {
  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <LineChart data={data}>
        <defs>
          <linearGradient id="colorGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="oklch(var(--primary))" stopOpacity={0.8}/>
            <stop offset="95%" stopColor="oklch(var(--primary))" stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="oklch(var(--border))" />
        <XAxis 
          dataKey="name" 
          stroke="oklch(var(--muted-foreground))"
          fontSize={12}
        />
        <YAxis 
          stroke="oklch(var(--muted-foreground))"
          fontSize={12}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'oklch(var(--card))',
            border: '1px solid oklch(var(--border))',
            borderRadius: '8px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
        />
        <Line
          type="monotone"
          dataKey="value"
          stroke="oklch(var(--primary))"
          strokeWidth={2}
          dot={{ fill: 'oklch(var(--primary))', strokeWidth: 2, r: 4 }}
          activeDot={{ r: 6, stroke: 'oklch(var(--primary))', strokeWidth: 2 }}
        />
      </LineChart>
    </ResponsiveContainer>
  ), [data, height])

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
  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="colorUv" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="oklch(var(--primary))" stopOpacity={0.6}/>
            <stop offset="95%" stopColor="oklch(var(--primary))" stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <XAxis 
          dataKey="name" 
          stroke="oklch(var(--muted-foreground))"
          fontSize={12}
        />
        <YAxis 
          stroke="oklch(var(--muted-foreground))"
          fontSize={12}
        />
        <CartesianGrid strokeDasharray="3 3" stroke="oklch(var(--border))" />
        <Tooltip
          contentStyle={{
            backgroundColor: 'oklch(var(--card))',
            border: '1px solid oklch(var(--border))',
            borderRadius: '8px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
        />
        <Area
          type="monotone"
          dataKey="value"
          stroke="oklch(var(--primary))"
          fillOpacity={1}
          fill="url(#colorUv)"
        />
      </AreaChart>
    </ResponsiveContainer>
  ), [data, height])

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
  const chartComponent = useMemo(() => (
    <ResponsiveContainer width="100%" height={height} debounce={100}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" stroke="oklch(var(--border))" />
        <XAxis 
          dataKey="name" 
          stroke="oklch(var(--muted-foreground))"
          fontSize={12}
        />
        <YAxis 
          stroke="oklch(var(--muted-foreground))"
          fontSize={12}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: 'oklch(var(--card))',
            border: '1px solid oklch(var(--border))',
            borderRadius: '8px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
        />
        <Bar 
          dataKey="value" 
          fill="oklch(var(--primary))"
          radius={[4, 4, 0, 0]}
        />
      </BarChart>
    </ResponsiveContainer>
  ), [data, height]);

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
              fill={COLORS[index % COLORS.length]} 
            />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            backgroundColor: 'oklch(var(--card))',
            border: '1px solid oklch(var(--border))',
            borderRadius: '8px',
            boxShadow: '0 4px 12px oklch(0 0 0 / 0.15)',
          }}
        />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  ), [data, height]);

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
