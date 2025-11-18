"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Upload, Download, Image, File, Trash2, MoreVertical, Eye, Link2 } from "lucide-react"
import { fileAPI } from "@/lib/api"
import { useState, useRef } from "react"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { cn } from "@/lib/utils"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { usePagination } from "@/hooks/use-pagination"
import { PageHeader } from "@/components/common/page-header"
import { FilterBar } from "@/components/common/filter-bar"

export default function FilesPage() {
  const {
    data: files,
    page,
    total,
    loading,
    search,
    setSearch,
    filters,
    updateFilter,
    setPage,
    refresh,
  } = usePagination({
    fetchFn: fileAPI.list,
    initialFilters: { category: "" },
  })
  
  const categoryFilter = filters.category || ""
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  
  // 上传对话框状态
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false)
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploadCategory, setUploadCategory] = useState("attachment")
  
  // URL上传对话框状态
  const [urlDialogOpen, setUrlDialogOpen] = useState(false)
  const [uploadUrl, setUploadUrl] = useState("")
  const [urlCategory, setUrlCategory] = useState("attachment")

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return "0 B"
    const k = 1024
    const sizes = ["B", "KB", "MB", "GB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  const getFileIcon = (fileType: string) => {
    if (fileType?.startsWith("image/")) return <Image className="h-5 w-5 text-blue-500" />
    return <File className="h-5 w-5 text-gray-500" />
  }

  const getCategoryText = (category: string) => {
    const categoryMap: Record<string, string> = {
      attachment: "附件",
      image: "图片",
      video: "视频",
      document: "文档",
      other: "其他",
    }
    return categoryMap[category] || category
  }

  const getFileTypeText = (mimeType: string) => {
    if (!mimeType) return "未知"
    const parts = mimeType.split("/")
    if (parts.length === 2) {
      return parts[1].toUpperCase()
    }
    return mimeType
  }

  // 文件上传
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploadFile(file)
    setUploadDialogOpen(true)
  }

  const handleUpload = async () => {
    if (!uploadFile) return

    try {
      setUploading(true)
      await fileAPI.upload(uploadFile, uploadCategory)
      toast.success("文件上传成功")
      setUploadDialogOpen(false)
      setUploadFile(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ""
      }
      refresh()
    } catch (error: any) {
      toast.error(error.message || "文件上传失败")
    } finally {
      setUploading(false)
    }
  }

  // URL上传
  const handleUploadFromURL = async () => {
    if (!uploadUrl) {
      toast.error("请输入文件URL")
      return
    }

    try {
      setUploading(true)
      await fileAPI.uploadFromURL(uploadUrl, urlCategory)
      toast.success("文件上传成功")
      setUrlDialogOpen(false)
      setUploadUrl("")
      refresh()
    } catch (error: any) {
      toast.error(error.message || "文件上传失败")
    } finally {
      setUploading(false)
    }
  }

  // 删除文件
  const handleDeleteFile = async (file: any) => {
    if (!confirm(`确定要删除文件 ${file.original_name} 吗？`)) {
      return
    }

    try {
      await fileAPI.delete(String(file.id))
      toast.success("文件删除成功")
      refresh()
    } catch (error: any) {
      toast.error(error.message || "删除文件失败")
    }
  }

  // 下载文件
  const handleDownloadFile = (file: any) => {
    window.open(fileAPI.download(String(file.id)), "_blank")
  }

  // 预览文件
  const handlePreviewFile = (file: any) => {
    if (file.file_type?.startsWith("image/")) {
      window.open(fileAPI.preview(String(file.id)), "_blank")
    } else {
      toast.info("该文件类型不支持预览")
    }
  }

  // 获取文件URL
  const handleGetFileURL = async (file: any) => {
    try {
      const response = await fileAPI.getURL(String(file.id))
      if (response.data) {
        const url = (response.data as any).url
        navigator.clipboard.writeText(url)
        toast.success("文件URL已复制到剪贴板")
      }
    } catch (error: any) {
      toast.error(error.message || "获取文件URL失败")
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="文件管理"
          description="上传和管理您的文件"
          actions={
            <>
              <Button variant="outline" onClick={() => setUrlDialogOpen(true)}>
                <Link2 className="h-4 w-4 mr-2" />
                从URL上传
              </Button>
              <Button onClick={() => fileInputRef.current?.click()}>
                <Upload className="h-4 w-4 mr-2" />
                上传文件
              </Button>
            </>
          }
        />

        {/* Filters */}
        <FilterBar
          search={search}
          onSearchChange={setSearch}
          searchPlaceholder="搜索文件名..."
          filters={
            <Select 
              value={categoryFilter || "all"} 
              onValueChange={(value) => updateFilter("category", value === "all" ? "" : value)}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="筛选分类" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部分类</SelectItem>
                <SelectItem value="attachment">附件</SelectItem>
                <SelectItem value="image">图片</SelectItem>
                <SelectItem value="video">视频</SelectItem>
                <SelectItem value="document">文档</SelectItem>
                <SelectItem value="other">其他</SelectItem>
              </SelectContent>
            </Select>
          }
        />

        {/* 文件数据表 */}
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[250px]">文件名</TableHead>
                <TableHead className="w-[120px]">分类</TableHead>
                <TableHead className="w-[100px]">大小</TableHead>
                <TableHead className="w-[180px]">上传时间</TableHead>
                <TableHead className="w-[180px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 5 }).map((_, index) => (
                  <TableRow key={index}>
                    <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                    <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                    <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                    <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                    <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                  </TableRow>
                ))
              ) : files.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="h-24 text-center">
                    暂无文件数据
                  </TableCell>
                </TableRow>
              ) : (
                files.map((record) => (
                  <TableRow key={record.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        {getFileIcon(record.file_type)}
                        <div className="flex-1 min-w-0">
                          <div className="font-medium truncate">{record.original_name}</div>
                          <div className="text-xs text-muted-foreground truncate">
                            {getFileTypeText(record.file_type)}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="text-xs">
                        {getCategoryText(record.category)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm text-muted-foreground">
                        {formatFileSize(record.file_size || 0)}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm text-muted-foreground">
                        {new Date(record.created_at).toLocaleString()}
                      </div>
                    </TableCell>
                    <TableCell>
                      <TooltipProvider>
                        <div className="flex items-center gap-1">
                          {/* 下载文件 */}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 hover:bg-green-50 text-green-600 hover:text-green-700"
                                onClick={() => handleDownloadFile(record)}
                              >
                                <Download className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-xs">下载文件</p>
                            </TooltipContent>
                          </Tooltip>

                          {/* 预览文件 (仅图片) */}
                          {record.file_type?.startsWith("image/") && (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-8 w-8 hover:bg-blue-50 text-blue-600 hover:text-blue-700"
                                  onClick={() => handlePreviewFile(record)}
                                >
                                  <Eye className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent side="top">
                                <p className="text-xs">预览图片</p>
                              </TooltipContent>
                            </Tooltip>
                          )}

                          {/* 复制URL */}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 hover:bg-purple-50 text-purple-600 hover:text-purple-700"
                                onClick={() => handleGetFileURL(record)}
                              >
                                <Link2 className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-xs">复制文件URL</p>
                            </TooltipContent>
                          </Tooltip>

                          {/* 删除文件 */}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 hover:bg-red-50 text-red-600 hover:text-red-700"
                                onClick={() => handleDeleteFile(record)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-xs">删除文件 (不可恢复)</p>
                            </TooltipContent>
                          </Tooltip>
                        </div>
                      </TooltipProvider>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* 分页 */}
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {total} 个文件，当前第 {page} 页
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

        {/* 隐藏的文件输入 */}
        <Input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleFileSelect}
        />

        {/* 上传文件对话框 */}
        <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>上传文件</DialogTitle>
              <DialogDescription>
                选择要上传的文件
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="upload-category">文件分类</Label>
                <Select
                  value={uploadCategory}
                  onValueChange={setUploadCategory}
                >
                  <SelectTrigger id="upload-category">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="attachment">附件</SelectItem>
                    <SelectItem value="image">图片</SelectItem>
                    <SelectItem value="video">视频</SelectItem>
                    <SelectItem value="document">文档</SelectItem>
                    <SelectItem value="other">其他</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {uploadFile && (
                <div className="p-4 border rounded-lg bg-muted/50">
                  <div className="flex items-center gap-3">
                    {getFileIcon(uploadFile.type)}
                    <div className="flex-1 min-w-0">
                      <div className="font-medium truncate">{uploadFile.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {formatFileSize(uploadFile.size)}
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setUploadDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleUpload} disabled={!uploadFile || uploading}>
                {uploading ? "上传中..." : "上传"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* URL上传对话框 */}
        <Dialog open={urlDialogOpen} onOpenChange={setUrlDialogOpen}>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>从URL上传</DialogTitle>
              <DialogDescription>
                通过URL上传文件
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="upload-url">文件URL *</Label>
                <Input
                  id="upload-url"
                  type="url"
                  value={uploadUrl}
                  onChange={(e) => setUploadUrl(e.target.value)}
                  placeholder="https://example.com/file.jpg"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="url-category">文件分类</Label>
                <Select
                  value={urlCategory}
                  onValueChange={setUrlCategory}
                >
                  <SelectTrigger id="url-category">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="attachment">附件</SelectItem>
                    <SelectItem value="image">图片</SelectItem>
                    <SelectItem value="video">视频</SelectItem>
                    <SelectItem value="document">文档</SelectItem>
                    <SelectItem value="other">其他</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setUrlDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleUploadFromURL} disabled={!uploadUrl || uploading}>
                {uploading ? "上传中..." : "上传"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </MainLayout>
  )
}
