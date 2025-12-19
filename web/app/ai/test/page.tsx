"use client"

import { useState } from "react"
import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Bot,
  Play,
  CheckCircle,
  XCircle,
  Loader2,
  Sparkles,
  MessageSquare,
  Brain,
  Zap,
} from "lucide-react"
import { aiAPI } from "@/lib/api"

interface TestResult {
  success: boolean
  result: any
  error: string | null
}

interface AITestResponse {
  service_status: string
  tests: {
    sentiment_analysis: TestResult
    keyword_extraction: TestResult
    ai_generation: TestResult
  }
}

export default function AITestPage() {
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<AITestResponse | null>(null)

  // 自定义生成测试
  const [generating, setGenerating] = useState(false)
  const [prompt, setPrompt] = useState("")
  const [generatedText, setGeneratedText] = useState("")

  // 情感分析测试
  const [analyzing, setAnalyzing] = useState(false)
  const [analyzeText, setAnalyzeText] = useState("")
  const [sentimentResult, setSentimentResult] = useState<any>(null)

  // 运行完整测试
  const handleRunTest = async () => {
    setTesting(true)
    setTestResult(null)
    try {
      const response = await aiAPI.test()
      if (response.code === 0 && response.data) {
        setTestResult(response.data as AITestResponse)
        toast.success("AI服务测试完成")
      } else {
        toast.error(response.msg || "测试失败")
      }
    } catch (error) {
      console.error("Test error:", error)
      toast.error("测试请求失败")
    } finally {
      setTesting(false)
    }
  }

  // 自定义生成
  const handleGenerate = async () => {
    if (!prompt.trim()) {
      toast.warning("请输入提示词")
      return
    }
    setGenerating(true)
    setGeneratedText("")
    try {
      const response = await aiAPI.generateGroupChat({
        group_name: "测试群组",
        group_topic: prompt,
        ai_persona: "友好的助手",
        response_type: "casual",
        max_length: 500,
        language: "zh",
      })
      if (response.code === 0 && response.data) {
        setGeneratedText((response.data as any).response || JSON.stringify(response.data))
        toast.success("生成成功")
      } else {
        toast.error(response.msg || "生成失败")
      }
    } catch (error) {
      console.error("Generate error:", error)
      toast.error("生成请求失败")
    } finally {
      setGenerating(false)
    }
  }

  // 情感分析
  const handleAnalyze = async () => {
    if (!analyzeText.trim()) {
      toast.warning("请输入要分析的文本")
      return
    }
    setAnalyzing(true)
    setSentimentResult(null)
    try {
      const response = await aiAPI.analyzeSentiment(analyzeText)
      if (response.code === 0 && response.data) {
        setSentimentResult(response.data)
        toast.success("分析完成")
      } else {
        toast.error(response.msg || "分析失败")
      }
    } catch (error) {
      console.error("Analyze error:", error)
      toast.error("分析请求失败")
    } finally {
      setAnalyzing(false)
    }
  }

  const StatusIcon = ({ success }: { success: boolean }) => {
    return success ? (
      <CheckCircle className="h-5 w-5 text-green-500" />
    ) : (
      <XCircle className="h-5 w-5 text-red-500" />
    )
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* 页面标题 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
              <Brain className="h-8 w-8 text-primary" />
              AI服务测试
            </h1>
            <p className="text-muted-foreground mt-1">
              测试AI服务连接和生成能力
            </p>
          </div>
        </div>

        <Tabs defaultValue="quick" className="space-y-6">
          <TabsList>
            <TabsTrigger value="quick" className="gap-2">
              <Zap className="h-4 w-4" />
              快速测试
            </TabsTrigger>
            <TabsTrigger value="generate" className="gap-2">
              <Sparkles className="h-4 w-4" />
              文本生成
            </TabsTrigger>
            <TabsTrigger value="sentiment" className="gap-2">
              <MessageSquare className="h-4 w-4" />
              情感分析
            </TabsTrigger>
          </TabsList>

          {/* 快速测试 */}
          <TabsContent value="quick" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Bot className="h-5 w-5" />
                  AI服务状态测试
                </CardTitle>
                <CardDescription>
                  一键测试AI服务的各项功能是否正常
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <Button
                  onClick={handleRunTest}
                  disabled={testing}
                  className="gap-2"
                  size="lg"
                >
                  {testing ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Play className="h-4 w-4" />
                  )}
                  {testing ? "测试中..." : "运行测试"}
                </Button>

                {testResult && (
                  <div className="mt-6 space-y-4">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">服务状态:</span>
                      <Badge
                        variant={
                          testResult.service_status === "available"
                            ? "default"
                            : testResult.service_status === "partial"
                            ? "secondary"
                            : "destructive"
                        }
                        className={
                          testResult.service_status === "available"
                            ? "bg-green-100 text-green-700"
                            : testResult.service_status === "partial"
                            ? "bg-yellow-100 text-yellow-700"
                            : ""
                        }
                      >
                        {testResult.service_status === "available"
                          ? "正常"
                          : testResult.service_status === "partial"
                          ? "部分可用"
                          : "不可用"}
                      </Badge>
                    </div>

                    <div className="grid gap-4 md:grid-cols-3">
                      {/* 情感分析测试结果 */}
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm flex items-center gap-2">
                            <StatusIcon success={testResult.tests.sentiment_analysis.success} />
                            情感分析
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          {testResult.tests.sentiment_analysis.success ? (
                            <div className="text-sm text-muted-foreground">
                              <p>情感: {testResult.tests.sentiment_analysis.result?.sentiment}</p>
                              <p>置信度: {(testResult.tests.sentiment_analysis.result?.confidence * 100).toFixed(0)}%</p>
                            </div>
                          ) : (
                            <p className="text-sm text-red-500">
                              {testResult.tests.sentiment_analysis.error}
                            </p>
                          )}
                        </CardContent>
                      </Card>

                      {/* 关键词提取测试结果 */}
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm flex items-center gap-2">
                            <StatusIcon success={testResult.tests.keyword_extraction.success} />
                            关键词提取
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          {testResult.tests.keyword_extraction.success ? (
                            <div className="text-sm text-muted-foreground">
                              <p>关键词: {testResult.tests.keyword_extraction.result?.join(", ") || "无"}</p>
                            </div>
                          ) : (
                            <p className="text-sm text-red-500">
                              {testResult.tests.keyword_extraction.error}
                            </p>
                          )}
                        </CardContent>
                      </Card>

                      {/* AI生成测试结果 */}
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="text-sm flex items-center gap-2">
                            <StatusIcon success={testResult.tests.ai_generation.success} />
                            AI生成 (Gemini)
                          </CardTitle>
                        </CardHeader>
                        <CardContent>
                          {testResult.tests.ai_generation.success ? (
                            <p className="text-sm text-muted-foreground line-clamp-3">
                              {testResult.tests.ai_generation.result}
                            </p>
                          ) : (
                            <p className="text-sm text-red-500">
                              {testResult.tests.ai_generation.error}
                            </p>
                          )}
                        </CardContent>
                      </Card>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* 文本生成 */}
          <TabsContent value="generate" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Sparkles className="h-5 w-5" />
                  AI文本生成
                </CardTitle>
                <CardDescription>
                  输入提示词，测试AI生成能力
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>提示词 / 话题</Label>
                  <Textarea
                    value={prompt}
                    onChange={(e) => setPrompt(e.target.value)}
                    placeholder="输入你想让AI讨论的话题，例如：最近的科技新闻、加密货币市场分析..."
                    rows={3}
                  />
                </div>

                <Button
                  onClick={handleGenerate}
                  disabled={generating}
                  className="gap-2"
                >
                  {generating ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Sparkles className="h-4 w-4" />
                  )}
                  {generating ? "生成中..." : "生成回复"}
                </Button>

                {generatedText && (
                  <div className="mt-4 p-4 bg-muted/50 rounded-lg">
                    <Label className="text-sm text-muted-foreground mb-2 block">生成结果:</Label>
                    <p className="whitespace-pre-wrap">{generatedText}</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* 情感分析 */}
          <TabsContent value="sentiment" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <MessageSquare className="h-5 w-5" />
                  情感分析
                </CardTitle>
                <CardDescription>
                  分析文本的情感倾向和关键词
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>待分析文本</Label>
                  <Textarea
                    value={analyzeText}
                    onChange={(e) => setAnalyzeText(e.target.value)}
                    placeholder="输入要分析的文本..."
                    rows={3}
                  />
                </div>

                <Button
                  onClick={handleAnalyze}
                  disabled={analyzing}
                  className="gap-2"
                >
                  {analyzing ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Brain className="h-4 w-4" />
                  )}
                  {analyzing ? "分析中..." : "开始分析"}
                </Button>

                {sentimentResult && (
                  <div className="mt-4 p-4 bg-muted/50 rounded-lg space-y-2">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">情感倾向:</span>
                      <Badge
                        variant={
                          sentimentResult.sentiment === "positive"
                            ? "default"
                            : sentimentResult.sentiment === "negative"
                            ? "destructive"
                            : "secondary"
                        }
                        className={
                          sentimentResult.sentiment === "positive"
                            ? "bg-green-100 text-green-700"
                            : ""
                        }
                      >
                        {sentimentResult.sentiment === "positive"
                          ? "积极"
                          : sentimentResult.sentiment === "negative"
                          ? "消极"
                          : "中性"}
                      </Badge>
                    </div>
                    <p><span className="font-medium">置信度:</span> {(sentimentResult.confidence * 100).toFixed(0)}%</p>
                    <p><span className="font-medium">意图:</span> {sentimentResult.intent}</p>
                    {sentimentResult.emotions?.length > 0 && (
                      <p><span className="font-medium">情绪:</span> {sentimentResult.emotions.join(", ")}</p>
                    )}
                    {sentimentResult.keywords?.length > 0 && (
                      <p><span className="font-medium">关键词:</span> {sentimentResult.keywords.join(", ")}</p>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </MainLayout>
  )
}
