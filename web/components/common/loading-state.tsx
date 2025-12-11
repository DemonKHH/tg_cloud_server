"use client"

import { motion } from "framer-motion"
import { cn } from "@/lib/utils"

interface LoadingStateProps {
  text?: string
  className?: string
  size?: "sm" | "md" | "lg"
}

export function LoadingState({ text = "加载中...", className, size = "md" }: LoadingStateProps) {
  const sizeClasses = {
    sm: "w-5 h-5 border-2",
    md: "w-8 h-8 border-2",
    lg: "w-12 h-12 border-3",
  }

  return (
    <div className={cn("flex flex-col items-center justify-center gap-3", className)}>
      <motion.div
        animate={{ rotate: 360 }}
        transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
        className={cn(
          "rounded-full border-primary border-t-transparent",
          sizeClasses[size]
        )}
      />
      {text && (
        <span className="text-sm text-muted-foreground">{text}</span>
      )}
    </div>
  )
}

// 表格骨架屏
interface TableSkeletonProps {
  rows?: number
  columns?: number
}

export function TableSkeleton({ rows = 5, columns = 6 }: TableSkeletonProps) {
  return (
    <>
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <tr key={rowIndex} className="border-b border-border/40">
          {Array.from({ length: columns }).map((_, colIndex) => (
            <td key={colIndex} className="px-3 py-3">
              <div className="h-4 bg-muted rounded animate-pulse" />
            </td>
          ))}
        </tr>
      ))}
    </>
  )
}

// 卡片骨架屏
export function CardSkeleton() {
  return (
    <div className="rounded-xl border border-border/50 p-5 space-y-3">
      <div className="flex items-center justify-between">
        <div className="h-4 w-24 bg-muted rounded animate-pulse" />
        <div className="h-8 w-8 bg-muted rounded-lg animate-pulse" />
      </div>
      <div className="h-8 w-16 bg-muted rounded animate-pulse" />
      <div className="h-3 w-20 bg-muted rounded animate-pulse" />
    </div>
  )
}
