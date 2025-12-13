import { useState, useEffect, useCallback, useRef } from "react"

interface UsePaginationOptions<T> {
  fetchFn: (params: any) => Promise<{ data?: any }>
  pageSize?: number
  initialFilters?: Record<string, any>
  debounceDelay?: number
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
  
  // 保存 fetchFn 的引用，避免依赖变化
  const fetchFnRef = useRef(fetchFn)
  fetchFnRef.current = fetchFn
  
  // 用于跟踪是否已经初始化
  const isInitialized = useRef(false)
  // 用于防抖
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  // 用于跟踪上一次的搜索值
  const lastSearchRef = useRef(search)

  const loadData = useCallback(async (currentPage: number, currentSearch: string, currentFilters: Record<string, any>) => {
    try {
      setLoading(true)
      const params: any = { page: currentPage, limit: pageSize }
      
      if (currentSearch) {
        params.search = currentSearch
      }
      
      // 合并所有过滤器
      Object.keys(currentFilters).forEach((key) => {
        if (currentFilters[key]) {
          params[key] = currentFilters[key]
        }
      })

      const response = await fetchFnRef.current(params)
      if (response.data) {
        const responseData = response.data as any
        setData(responseData.items || responseData.data || [])
        setTotal(responseData.pagination?.total || responseData.total || 0)
      }
    } catch (error) {
      console.error("Failed to load data:", error)
    } finally {
      setLoading(false)
    }
  }, [pageSize])

  // 初始加载
  useEffect(() => {
    if (!isInitialized.current) {
      isInitialized.current = true
      loadData(page, search, filters)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // 页面/过滤器变化时重新加载（跳过初始化）
  useEffect(() => {
    if (!isInitialized.current) return
    loadData(page, search, filters)
  }, [page, filters]) // eslint-disable-line react-hooks/exhaustive-deps

  // 搜索防抖
  useEffect(() => {
    // 跳过初始化
    if (!isInitialized.current) return
    // 跳过相同的搜索值
    if (lastSearchRef.current === search) return

    // 清除之前的 timer
    if (searchTimerRef.current) {
      clearTimeout(searchTimerRef.current)
    }

    searchTimerRef.current = setTimeout(() => {
      lastSearchRef.current = search
      // 搜索变化时总是加载第一页的数据
      if (page !== 1) {
        setPage(1) // 这会触发上面的 useEffect，但 search 已经是新值了
      } else {
        loadData(1, search, filters)
      }
    }, debounceDelay)

    return () => {
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current)
      }
    }
  }, [search, debounceDelay]) // eslint-disable-line react-hooks/exhaustive-deps

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
    loadData(page, search, filters)
  }, [loadData, page, search, filters])

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
