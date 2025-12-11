"use client"

import { motion } from "framer-motion"
import { ReactNode } from "react"

interface PageHeaderProps {
  title: string
  description?: string
  actions?: ReactNode
}

export function PageHeader({ title, description, actions }: PageHeaderProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
      className="flex flex-col gap-1 md:flex-row md:items-center md:justify-between"
    >
      <div>
        <h1 className="page-title gradient-text">{title}</h1>
        {description && (
          <p className="page-subtitle">{description}</p>
        )}
      </div>
      {actions && (
        <div className="flex flex-wrap items-center gap-2 mt-3 md:mt-0">
          {actions}
        </div>
      )}
    </motion.div>
  )
}
