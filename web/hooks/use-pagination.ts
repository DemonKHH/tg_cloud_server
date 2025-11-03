import { useState, useEffect, useCallback } from "react"

interface UsePaginationOptions<T> {
  fetchFn: (params: any) => Promise<{ data?: any }>
  pageSize?: number
  initialFilters?: Record<string, any>
  debounceDelay?: number
}

interface PaginationState {
  page: number
  total: number
  loading: boolean
  search: string
  filters: Record<string, any>
}

export function usePagination<T = any>(options: UsePaginationOptions<T>) {
  const {
    fetchFn,
    pageSize = 20,
    initialFilters = {},
    debounceDelay = 500,
  } = options

  const [data, setData] = useState<T[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")
  const [filters, setFilters] = useState<Record<string, any>>(initialFilters)

  const loadData = useCallback(async () => {
    try {
      setLoading(true)
      const params: any = { page, limit: pageSize }
      
      if (search) {
        params.search = search
      }
      
      // 合并所有过滤器
      Object.keys(filters).forEach((key) => {
        if (filters[key]) {
          params[key] = filters[key]
        }
      })

      const response = await fetchFn(params)
      if (response.data) {
        const responseData = response.data as any
        setData(responseData.items || responseData.data || [])
        setTotal(responseData.pagination?.total || responseData.total || 0)
        
        // 同步页面号（如果后端返回了当前页）
        if (responseData.pagination?.current_page) {
          setPage(responseData.pagination.current_page)
        }
      }
    } catch (error) {
      console.error("Failed to load data:", error)
      throw error
    } finally {
      setLoading(false)
    }
  }, [fetchFn, page, search, filters, pageSize])

  // 初始加载和页面/过滤器变化时重新加载
  useEffect(() => {
    loadData()
  }, [page, filters]) // search 通过 debounce 处理，不直接依赖

  // 搜索防抖
  useEffect(() => {
    const timer = setTimeout(() => {
      if (page === 1) {
        loadData()
      } else {
        setPage(1) // 重置到第一页，触发 loadData
      }
    }, debounceDelay)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  const updateFilter = useCallback((key: string, value: any) => {
    setFilters((prev) => ({ ...prev, [key]: value }))
    setPage(1) // 重置到第一页
  }, [])

  const resetFilters = useCallback(() => {
    setFilters(initialFilters)
    setSearch("")
    setPage(1)
  }, [initialFilters])

  const refresh = useCallback(() => {
    loadData()
  }, [loadData])

  return {
    data,
    page,
    total,
    loading,
    search,
    setSearch,
    filters,
    setFilters,
    updateFilter,
    resetFilters,
    setPage,
    refresh,
  }
}

