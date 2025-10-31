"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, Search, Filter, MoreVertical, CheckCircle2, XCircle, AlertCircle } from "lucide-react"
import { accountAPI } from "@/lib/api"
import { useState, useEffect } from "react"

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState("")

  useEffect(() => {
    loadAccounts()
  }, [page])

  const loadAccounts = async () => {
    try {
      setLoading(true)
      const response = await accountAPI.list({ page, limit: 20 })
      if (response.data) {
        setAccounts(response.data.data || [])
        setTotal(response.data.total || 0)
      }
    } catch (error) {
      toast.error("加载账号失败，请稍后重试")
      console.error("加载账号失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case "error":
        return <XCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "active":
        return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300"
      case "error":
        return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300"
      default:
        return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300"
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">账号管理</h1>
            <p className="text-muted-foreground mt-1">
              管理和监控您的TG账号
            </p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            添加账号
          </Button>
        </div>

        {/* Filters */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  placeholder="搜索手机号或备注..."
                  className="pl-9"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>
              <Button variant="outline">
                <Filter className="h-4 w-4 mr-2" />
                筛选
              </Button>
            </div>
          </CardHeader>
        </Card>

        {/* Accounts Table */}
        <Card>
          <CardHeader>
            <CardTitle>账号列表</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">加载中...</div>
            ) : accounts.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">暂无账号</div>
            ) : (
              <div className="space-y-4">
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b">
                        <th className="text-left p-4 font-medium">手机号</th>
                        <th className="text-left p-4 font-medium">状态</th>
                        <th className="text-left p-4 font-medium">健康度</th>
                        <th className="text-left p-4 font-medium">代理</th>
                        <th className="text-left p-4 font-medium">最后使用</th>
                        <th className="text-right p-4 font-medium">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {accounts.map((account) => (
                        <tr key={account.id} className="border-b hover:bg-muted/50">
                          <td className="p-4">
                            <div className="font-medium">{account.phone}</div>
                            {account.note && (
                              <div className="text-sm text-muted-foreground">{account.note}</div>
                            )}
                          </td>
                          <td className="p-4">
                            <div className="flex items-center gap-2">
                              {getStatusIcon(account.status)}
                              <span
                                className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(
                                  account.status
                                )}`}
                              >
                                {account.status}
                              </span>
                            </div>
                          </td>
                          <td className="p-4">
                            <div className="flex items-center gap-2">
                              <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden max-w-24">
                                <div
                                  className="h-full bg-green-500"
                                  style={{ width: `${(account.health_score || 0) * 100}%` }}
                                />
                              </div>
                              <span className="text-sm font-medium">
                                {((account.health_score || 0) * 100).toFixed(0)}%
                              </span>
                            </div>
                          </td>
                          <td className="p-4">
                            <div className="text-sm">
                              {account.proxy_id ? (
                                <span className="text-muted-foreground">已绑定</span>
                              ) : (
                                <span className="text-muted-foreground">未绑定</span>
                              )}
                            </div>
                          </td>
                          <td className="p-4">
                            <div className="text-sm text-muted-foreground">
                              {account.last_used_at
                                ? new Date(account.last_used_at).toLocaleDateString()
                                : "从未使用"}
                            </div>
                          </td>
                          <td className="p-4 text-right">
                            <Button variant="ghost" size="icon">
                              <MoreVertical className="h-4 w-4" />
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                <div className="flex items-center justify-between pt-4">
                  <div className="text-sm text-muted-foreground">
                    共 {total} 个账号
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => Math.max(1, p - 1))}
                      disabled={page === 1}
                    >
                      上一页
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage((p) => p + 1)}
                      disabled={page * 20 >= total}
                    >
                      下一页
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </MainLayout>
  )
}

