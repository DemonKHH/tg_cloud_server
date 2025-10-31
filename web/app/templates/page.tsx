"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, Search, FileText, Copy, Trash2 } from "lucide-react"
import { templateAPI } from "@/lib/api"
import { useState, useEffect } from "react"

export default function TemplatesPage() {
  const [templates, setTemplates] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadTemplates()
  }, [])

  const loadTemplates = async () => {
    try {
      setLoading(true)
      const response = await templateAPI.list({ page: 1, limit: 20 })
      if (response.data) {
        setTemplates(response.data.data || [])
      }
    } catch (error) {
      toast.error("加载模板失败，请稍后重试")
      console.error("加载模板失败:", error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">模板管理</h1>
            <p className="text-muted-foreground mt-1">管理和使用消息模板</p>
          </div>
          <Button>
            <Plus className="h-4 w-4 mr-2" />
            创建模板
          </Button>
        </div>

        <Card>
          <CardHeader>
            <div className="flex items-center gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  placeholder="搜索模板..."
                  className="pl-9"
                />
              </div>
            </div>
          </CardHeader>
        </Card>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {loading ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              加载中...
            </div>
          ) : templates.length === 0 ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              暂无模板
            </div>
          ) : (
            templates.map((template) => (
              <Card key={template.id} className="hover:shadow-md transition-shadow">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <CardTitle className="text-lg">{template.name}</CardTitle>
                      <div className="mt-1 text-sm text-muted-foreground">
                        {template.type} · {template.category}
                      </div>
                    </div>
                    <FileText className="h-5 w-5 text-muted-foreground" />
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {template.content}
                    </p>
                    <div className="flex gap-2">
                      <Button variant="outline" size="sm" className="flex-1">
                        使用
                      </Button>
                      <Button variant="outline" size="icon">
                        <Copy className="h-4 w-4" />
                      </Button>
                      <Button variant="outline" size="icon">
                        <Trash2 className="h-4 w-4" />
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

