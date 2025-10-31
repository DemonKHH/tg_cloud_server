"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Plus, TestTube, CheckCircle2, XCircle } from "lucide-react"
import { proxyAPI } from "@/lib/api"
import { useState, useEffect } from "react"

export default function ProxiesPage() {
  const [proxies, setProxies] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadProxies()
  }, [])

  const loadProxies = async () => {
    try {
      setLoading(true)
      const response = await proxyAPI.list({ page: 1, limit: 50 })
      if (response.data) {
        // 后端返回格式：{ items: [], pagination: { total, current_page, ... } }
        const data = response.data as any
        setProxies(data.items || [])
      }
    } catch (error) {
      toast.error("加载代理失败，请稍后重试")
      console.error("加载代理失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const handleTest = async (id: string) => {
    try {
      const response = await proxyAPI.test(id)
      toast.success("代理测试成功")
      loadProxies()
    } catch (error) {
      toast.error("代理测试失败")
      console.error("测试代理失败:", error)
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">代理管理</h1>
            <p className="text-muted-foreground mt-1">管理您的代理IP配置</p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            添加代理
          </Button>
        </div>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {loading ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              加载中...
            </div>
          ) : proxies.length === 0 ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              暂无代理
            </div>
          ) : (
            proxies.map((proxy) => (
              <Card key={proxy.id} className="hover:shadow-md transition-shadow">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <CardTitle className="text-lg">
                        {proxy.host}:{proxy.port}
                      </CardTitle>
                      <div className="mt-1 text-sm text-muted-foreground">
                        {proxy.protocol?.toUpperCase()}
                      </div>
                    </div>
                    {proxy.status === "active" ? (
                      <CheckCircle2 className="h-5 w-5 text-green-500" />
                    ) : (
                      <XCircle className="h-5 w-5 text-red-500" />
                    )}
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">状态:</span>
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-medium ${
                          proxy.status === "active"
                            ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300"
                            : "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300"
                        }`}
                      >
                        {proxy.status}
                      </span>
                    </div>
                    {proxy.username && (
                      <div className="flex items-center justify-between text-sm">
                        <span className="text-muted-foreground">用户名:</span>
                        <span className="font-mono">{proxy.username}</span>
                      </div>
                    )}
                    <div className="flex gap-2 pt-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="flex-1"
                        onClick={() => handleTest(String(proxy.id))}
                      >
                        <TestTube className="h-4 w-4 mr-1" />
                        测试
                      </Button>
                      <Button variant="outline" size="sm" className="flex-1">
                        编辑
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      </div>
    </MainLayout>
  )
}

