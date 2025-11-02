"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, Search, Play, Pause, X, RefreshCw } from "lucide-react"
import { taskAPI } from "@/lib/api"
import { useState, useEffect } from "react"

export default function TasksPage() {
  const [tasks, setTasks] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  useEffect(() => {
    loadTasks()
  }, [page])

  const loadTasks = async () => {
    try {
      setLoading(true)
      const response = await taskAPI.list({ page, limit: 20 })
      if (response.data) {
        // 后端返回格式：{ items: [], pagination: { total, current_page, ... } }
        const data = response.data as any
        setTasks(data.items || [])
      }
    } catch (error) {
      toast.error("加载任务失败，请稍后重试")
      console.error("加载任务失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-50 text-green-700 border border-green-200 dark:bg-green-900 dark:text-green-300 dark:border-green-800"
      case "running":
        return "bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800"
      case "failed":
        return "bg-red-50 text-red-700 border border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800"
      case "queued":
        return "bg-yellow-50 text-yellow-700 border border-yellow-200 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800"
      default:
        return "bg-gray-50 text-gray-700 border border-gray-200 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-800"
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">任务管理</h1>
            <p className="text-muted-foreground mt-1">
              查看和管理您的任务
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline">
              <RefreshCw className="h-4 w-4 mr-2" />
              刷新
            </Button>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              创建任务
            </Button>
          </div>
        </div>

        {/* Tasks List */}
        <div className="grid gap-4">
          {loading ? (
            <Card className="card-shadow">
              <CardContent className="py-8 text-center text-muted-foreground">
                加载中...
              </CardContent>
            </Card>
          ) : tasks.length === 0 ? (
            <Card className="card-shadow">
              <CardContent className="py-8 text-center text-muted-foreground">
                暂无任务
              </CardContent>
            </Card>
          ) : (
            tasks.map((task) => (
              <Card key={task.id} className="card-shadow hover:card-shadow-lg transition-all duration-300">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <CardTitle className="text-lg">
                          {task.task_type || "未知任务"}
                        </CardTitle>
                        <span
                          className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(
                            task.status
                          )}`}
                        >
                          {task.status}
                        </span>
                      </div>
                      <div className="mt-2 text-sm text-muted-foreground">
                        账号ID: {task.account_id} | 优先级: {task.priority}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      {task.status === "queued" && (
                        <Button variant="outline" size="sm">
                          <Play className="h-4 w-4 mr-1" />
                          开始
                        </Button>
                      )}
                      {task.status === "running" && (
                        <Button variant="outline" size="sm">
                          <Pause className="h-4 w-4 mr-1" />
                          暂停
                        </Button>
                      )}
                      <Button variant="outline" size="sm">
                        <X className="h-4 w-4 mr-1" />
                        取消
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-3">
                    <div>
                      <div className="text-xs text-muted-foreground mb-1">创建时间</div>
                      <div className="text-sm">
                        {new Date(task.created_at).toLocaleString()}
                      </div>
                    </div>
                    <div>
                      <div className="text-xs text-muted-foreground mb-1">更新时间</div>
                      <div className="text-sm">
                        {task.updated_at
                          ? new Date(task.updated_at).toLocaleString()
                          : "-"}
                      </div>
                    </div>
                    <div>
                      <div className="text-xs text-muted-foreground mb-1">完成时间</div>
                      <div className="text-sm">
                        {task.completed_at
                          ? new Date(task.completed_at).toLocaleString()
                          : "-"}
                      </div>
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

