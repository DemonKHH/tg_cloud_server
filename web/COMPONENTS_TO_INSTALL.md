# shadcn/ui 组件安装清单

根据已创建的页面和功能需求，以下是需要安装的组件：

## 🔴 必需组件（核心功能）

### 1. **Table** - 表格组件
- **用途**: 账号列表、任务列表等数据展示
- **页面**: `accounts/page.tsx`, `tasks/page.tsx`
- **安装命令**: 
```bash
npx shadcn@latest add table
```

### 2. **Dialog** - 对话框/弹窗
- **用途**: 创建/编辑账号、任务、代理的表单弹窗
- **页面**: 所有管理页面都需要
- **安装命令**:
```bash
npx shadcn@latest add dialog
```

### 3. **Select** - 下拉选择框
- **用途**: 状态筛选、类型选择、代理选择等
- **页面**: 所有筛选和表单页面
- **安装命令**:
```bash
npx shadcn@latest add select
```

### 4. **Badge** - 标签/徽章
- **用途**: 状态显示（active, error, completed等）
- **页面**: `accounts/page.tsx`, `tasks/page.tsx`, `proxies/page.tsx`
- **安装命令**:
```bash
npx shadcn@latest add badge
```

### 5. **Dropdown Menu** - 下拉菜单
- **用途**: 操作菜单（编辑、删除、更多操作）
- **页面**: 所有列表页面
- **安装命令**:
```bash
npx shadcn@latest add dropdown-menu
```

### 6. **Toast** 或 **Sonner** - 消息提示
- **用途**: 成功/错误消息提示
- **页面**: 所有操作页面
- **推荐使用 Sonner**（更现代化）
- **安装命令**:
```bash
npx shadcn@latest add sonner
```

### 7. **Skeleton** - 骨架屏
- **用途**: 加载状态占位符
- **页面**: 所有数据加载页面
- **安装命令**:
```bash
npx shadcn@latest add skeleton
```

### 8. **Alert** - 警告提示
- **用途**: 错误提示、警告信息
- **页面**: 登录页、表单验证等
- **安装命令**:
```bash
npx shadcn@latest add alert
```

### 9. **Label** - 表单标签
- **用途**: 表单输入标签
- **页面**: 所有表单页面
- **安装命令**:
```bash
npx shadcn@latest add label
```

### 10. **Textarea** - 多行文本输入
- **用途**: 模板内容、备注等长文本输入
- **页面**: `templates/page.tsx`, 各种表单
- **安装命令**:
```bash
npx shadcn@latest add textarea
```

## 🟡 推荐组件（提升体验）

### 11. **Tabs** - 选项卡
- **用途**: 详情页切换、数据分类展示
- **页面**: 账号详情、任务详情等
- **安装命令**:
```bash
npx shadcn@latest add tabs
```

### 12. **Pagination** - 分页组件
- **用途**: 列表分页（目前用了简单按钮）
- **页面**: 所有列表页面
- **安装命令**:
```bash
npx shadcn@latest add pagination
```

### 13. **Progress** - 进度条
- **用途**: 健康度显示、任务进度
- **页面**: `accounts/page.tsx`, `dashboard/page.tsx`
- **安装命令**:
```bash
npx shadcn@latest add progress
```

### 14. **Separator** - 分隔线
- **用途**: 内容分隔
- **安装命令**:
```bash
npx shadcn@latest add separator
```

### 15. **Switch** - 开关
- **用途**: 状态切换、功能开关
- **安装命令**:
```bash
npx shadcn@latest add switch
```

### 16. **Tooltip** - 提示工具
- **用途**: 操作提示、说明信息
- **安装命令**:
```bash
npx shadcn@latest add tooltip
```

## 🟢 可选组件（高级功能）

### 17. **Sheet** - 侧边抽屉
- **用途**: 移动端侧边菜单、详情抽屉
- **安装命令**:
```bash
npx shadcn@latest add sheet
```

### 18. **Popover** - 弹出框
- **用途**: 浮动操作菜单、详细信息
- **安装命令**:
```bash
npx shadcn@latest add popover
```

### 19. **Checkbox** - 复选框
- **用途**: 批量选择、多选功能
- **安装命令**:
```bash
npx shadcn@latest add checkbox
```

### 20. **Radio Group** - 单选框组
- **用途**: 单选操作
- **安装命令**:
```bash
npx shadcn@latest add radio-group
```

---

## 📦 批量安装命令

### 核心组件（一次安装）
```bash
cd web
npx shadcn@latest add table dialog select badge dropdown-menu sonner skeleton alert label textarea
```

### 推荐组件
```bash
npx shadcn@latest add tabs pagination progress separator switch tooltip
```

### 全部组件
```bash
npx shadcn@latest add table dialog select badge dropdown-menu sonner skeleton alert label textarea tabs pagination progress separator switch tooltip sheet popover checkbox radio-group
```

---

## 📝 使用示例

安装后，这些组件将自动添加到 `components/ui/` 目录，可以直接导入使用：

```tsx
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"
```

---

## ⚠️ 注意事项

1. **Sonner** 需要额外配置 Provider，需要在 `layout.tsx` 中添加 `<Toaster />`
2. **Dialog** 依赖 `@radix-ui/react-dialog`，会自动安装依赖
3. 所有组件都会自动适配 dark mode（已配置）
4. 组件样式已根据 `components.json` 中的配置自动应用

