"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Bot, Sparkles, MessageSquare, Brain } from "lucide-react"
import { aiAPI } from "@/lib/api"
import { useState } from "react"

export default function AIPage() {
  const [text, setText] = useState("")
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  const handleAnalyzeSentiment = async () => {
    if (!text.trim()) {
      toast.warning("请输入要分析的文本")
      return
    }
    try {
      setLoading(true)
      const response = await aiAPI.analyzeSentiment(text)
      if (response.data) {
        setResult(response.data)
        toast.success("情感分析完成")
      }
    } catch (error) {
      toast.error("情感分析失败")
      console.error("分析失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const handleExtractKeywords = async () => {
    if (!text.trim()) {
      toast.warning("请输入要提取关键词的文本")
      return
    }
    try {
      setLoading(true)
      const response = await aiAPI.extractKeywords(text)
      if (response.data) {
        setResult(response.data)
        toast.success("关键词提取完成")
      }
    } catch (error) {
      toast.error("关键词提取失败")
      console.error("提取失败:", error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">AI服务</h1>
          <p className="text-muted-foreground mt-1">
            使用AI分析文本、生成内容等
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MessageSquare className="h-5 w-5" />
                情感分析
              </CardTitle>
              <CardDescription>分析文本的情感倾向</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Input
                placeholder="输入要分析的文本..."
                value={text}
                onChange={(e) => setText(e.target.value)}
              />
              <Button onClick={handleAnalyzeSentiment} disabled={loading || !text.trim()}>
                <Sparkles className="h-4 w-4 mr-2" />
                {loading ? "分析中..." : "开始分析"}
              </Button>
              {result && (
                <div className="p-4 bg-muted rounded-lg">
                  <div className="text-sm font-medium mb-2">分析结果</div>
                  <div className="text-sm text-muted-foreground">
                    情感: {result.sentiment} | 置信度: {result.confidence}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Brain className="h-5 w-5" />
                关键词提取
              </CardTitle>
              <CardDescription>从文本中提取关键词</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Input
                placeholder="输入要提取关键词的文本..."
                value={text}
                onChange={(e) => setText(e.target.value)}
              />
              <Button onClick={handleExtractKeywords} disabled={loading || !text.trim()}>
                <Sparkles className="h-4 w-4 mr-2" />
                {loading ? "提取中..." : "提取关键词"}
              </Button>
              {result && result.keywords && (
                <div className="p-4 bg-muted rounded-lg">
                  <div className="text-sm font-medium mb-2">关键词</div>
                  <div className="flex flex-wrap gap-2">
                    {result.keywords.map((keyword: string, idx: number) => (
                      <span
                        key={idx}
                        className="px-2 py-1 bg-background rounded text-sm"
                      >
                        {keyword}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        <Card className="card-shadow">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Bot className="h-5 w-5" />
              AI功能说明
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              <div className="p-4 border rounded-lg bg-muted/30 hover:bg-muted/50 transition-colors">
                <div className="font-medium mb-1">群聊AI回复</div>
                <div className="text-sm text-muted-foreground">
                  根据群聊历史生成智能回复
                </div>
              </div>
              <div className="p-4 border rounded-lg bg-muted/30 hover:bg-muted/50 transition-colors">
                <div className="font-medium mb-1">私信内容生成</div>
                <div className="text-sm text-muted-foreground">
                  生成个性化私信内容
                </div>
              </div>
              <div className="p-4 border rounded-lg bg-muted/30 hover:bg-muted/50 transition-colors">
                <div className="font-medium mb-1">模板变体生成</div>
                <div className="text-sm text-muted-foreground">
                  生成多个模板变体版本
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </MainLayout>
  )
}

