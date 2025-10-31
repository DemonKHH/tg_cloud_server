"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, Upload, Download, Image, File, Trash2 } from "lucide-react"
import { fileAPI } from "@/lib/api"
import { useState, useEffect } from "react"

export default function FilesPage() {
  const [files, setFiles] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadFiles()
  }, [])

  const loadFiles = async () => {
    try {
      setLoading(true)
      const response = await fileAPI.list({ page: 1, limit: 20 })
      if (response.data) {
        setFiles(response.data.data || [])
      }
    } catch (error) {
      toast.error("加载文件失败，请稍后重试")
      console.error("加载文件失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    try {
      await fileAPI.upload(file, "attachment")
      toast.success("文件上传成功")
      loadFiles()
    } catch (error) {
      toast.error("文件上传失败")
      console.error("上传失败:", error)
    }
  }

  const getFileIcon = (fileType: string) => {
    if (fileType === "image") return <Image className="h-5 w-5" />
    return <File className="h-5 w-5" />
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">文件管理</h1>
            <p className="text-muted-foreground mt-1">上传和管理您的文件</p>
          </div>
          <div>
            <Input
              type="file"
              id="file-upload"
              className="hidden"
              onChange={handleUpload}
            />
            <Button asChild>
              <label htmlFor="file-upload" className="cursor-pointer">
                <Upload className="h-4 w-4 mr-2" />
                上传文件
              </label>
            </Button>
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {loading ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              加载中...
            </div>
          ) : files.length === 0 ? (
            <div className="col-span-full text-center py-8 text-muted-foreground">
              暂无文件
            </div>
          ) : (
            files.map((file) => (
              <Card key={file.id} className="hover:shadow-md transition-shadow">
                <CardHeader className="p-4">
                  <div className="flex items-center gap-3">
                    {getFileIcon(file.file_type)}
                    <div className="flex-1 min-w-0">
                      <CardTitle className="text-sm truncate">{file.original_name}</CardTitle>
                      <div className="text-xs text-muted-foreground mt-1">
                        {(file.file_size / 1024).toFixed(1)} KB
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="p-4 pt-0">
                  <div className="flex gap-2">
                    <Button variant="outline" size="sm" className="flex-1">
                      <Download className="h-3 w-3 mr-1" />
                      下载
                    </Button>
                    <Button variant="outline" size="icon">
                      <Trash2 className="h-3 w-3" />
                    </Button>
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

