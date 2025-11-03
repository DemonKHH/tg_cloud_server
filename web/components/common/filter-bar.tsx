import { Input } from "@/components/ui/input"
import { Card, CardHeader } from "@/components/ui/card"
import { Search } from "lucide-react"
import { ReactNode } from "react"

interface FilterBarProps {
  search: string
  onSearchChange: (value: string) => void
  searchPlaceholder?: string
  filters?: ReactNode
  className?: string
}

export function FilterBar({
  search,
  onSearchChange,
  searchPlaceholder = "搜索...",
  filters,
  className,
}: FilterBarProps) {
  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center gap-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              placeholder={searchPlaceholder}
              className="pl-9"
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
            />
          </div>
          {filters}
        </div>
      </CardHeader>
    </Card>
  )
}

