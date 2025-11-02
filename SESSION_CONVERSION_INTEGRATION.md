# Session/TData 转 SessionString 集成指南

## 📋 目录
1. [依赖库](#依赖库)
2. [数据结构](#数据结构)
3. [核心方法清单](#核心方法清单)
4. [方法详细实现](#方法详细实现)
5. [使用示例](#使用示例)

---

## 🔧 依赖库

### Go 模块依赖
```go
require (
    github.com/gotd/td v0.131.0              // Telegram客户端库
    github.com/mattn/go-sqlite3 v1.14.32     // SQLite驱动（用于.session文件）
)

import (
    "context"
    "crypto/sha1"
    "database/sql"
    "encoding/base64"
    "encoding/hex"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "strings"
    
    "github.com/gotd/td/session"
    "github.com/gotd/td/session/tdesktop"
    _ "github.com/mattn/go-sqlite3"
)
```

---

## 📦 数据结构

### SessionData 结构体
```go
type SessionData struct {
    EncodedData string // 编码后的会话数据（base64编码的SessionString）
    Username    string
    FirstName   string
    LastName    string
    UserID      int64
    IsPremium   bool
    Phone       string
    AuthKey     []byte // 原始认证密钥（256字节）
    DataCenter  int    // 数据中心ID (1-5)
}
```

---

## 📝 核心方法清单

### 🔹 Session转SessionString方法

| 方法名 | 作用 | 依赖 |
|--------|------|------|
| `loadPyrogramSession` | 加载.session文件 | - |
| `parseSessionDatabase` | 解析SQLite数据库 | `getTableInfo`, `buildSessionQuery`, `processAuthKey`, `loadUserInfo` |
| `convertPyrogramToGotd` | 转换Pyrogram格式为gotd格式 | `calculateAuthKeyID` |
| `calculateAuthKeyID` | 计算auth_key_id | - |
| `processAuthKey` | 处理auth_key数据 | `min` |
| `buildSessionQuery` | 构建SQL查询语句 | `getTableColumns`, `hasColumn` |
| `getTableInfo` | 获取数据库表信息 | - |
| `getTableColumns` | 获取表的列信息 | - |
| `hasColumn` | 检查列是否存在 | - |
| `loadUserInfo` | 加载用户信息 | `getTableInfo`, `hasColumn` |
| `min` | 返回较小值 | - |

### 🔹 TData转SessionString方法

| 方法名 | 作用 | 依赖 |
|--------|------|------|
| `loadTDataSession` | 加载tdata文件夹 | `session.TDesktopSession`, `tdesktop.Read` |
| `isTDataDirectory` | 检查是否为tdata文件夹 | - |

### 🔹 通用方法

| 方法名 | 作用 |
|--------|------|
| `loadSessionFromFiles` | 从文件加载会话数据（自动识别格式） |

---

## 🔨 方法详细实现

### 1. Session转SessionString流程

#### 1.1 loadPyrogramSession - 主入口
```go
func loadPyrogramSession(sessionFile, phone string) (*SessionData, error) {
    // 打开SQLite数据库
    db, err := sql.Open("sqlite3", sessionFile)
    if err != nil {
        return nil, fmt.Errorf("打开session数据库失败: %w", err)
    }
    defer db.Close()
    
    // 解析session数据
    return parseSessionDatabase(db, phone)
}
```

#### 1.2 parseSessionDatabase - 解析数据库
```go
func parseSessionDatabase(db *sql.DB, phone string) (*SessionData, error) {
    sessionData := &SessionData{Phone: phone}
    
    // 构建查询语句
    query, err := buildSessionQuery(db)
    if err != nil {
        return nil, fmt.Errorf("构建查询语句失败: %w", err)
    }
    
    // 查询sessions表
    var dcID int
    var authKey []byte
    var userID int64
    var isBot bool
    var authKeyData interface{}
    
    row := db.QueryRow(query)
    err = row.Scan(&dcID, &authKeyData, &userID, &isBot)
    if err != nil {
        return nil, fmt.Errorf("查询session信息失败: %w", err)
    }
    
    // 处理auth_key
    authKey, err = processAuthKey(authKeyData)
    if err != nil {
        return nil, fmt.Errorf("处理auth_key失败: %w", err)
    }
    
    sessionData.DataCenter = dcID
    sessionData.AuthKey = authKey
    sessionData.UserID = userID
    
    // 加载用户信息（可选）
    if userID > 0 {
        loadUserInfo(db, userID, sessionData)
    }
    
    // 转换为gotd格式
    tempSessionData := &SessionData{
        AuthKey:    authKey,
        DataCenter: dcID,
    }
    
    storage, err := convertPyrogramToGotd(tempSessionData)
    if err != nil {
        return nil, fmt.Errorf("转换session格式失败: %w", err)
    }
    
    // 获取二进制数据并编码
    ctx := context.Background()
    sessionBytes, err := storage.LoadSession(ctx)
    if err != nil {
        return nil, fmt.Errorf("获取session数据失败: %w", err)
    }
    
    if len(sessionBytes) == 0 {
        return nil, fmt.Errorf("session数据为空")
    }
    
    // Base64编码为SessionString
    sessionData.EncodedData = base64.StdEncoding.EncodeToString(sessionBytes)
    
    return sessionData, nil
}
```

#### 1.3 convertPyrogramToGotd - 格式转换
```go
func convertPyrogramToGotd(sessionData *SessionData) (*session.StorageMemory, error) {
    // 验证auth_key长度
    if len(sessionData.AuthKey) != 256 {
        return nil, fmt.Errorf("invalid auth_key length: %d, expected 256", len(sessionData.AuthKey))
    }
    
    // 创建内存存储
    storage := new(session.StorageMemory)
    loader := session.Loader{Storage: storage}
    
    // 根据DC ID确定服务器地址
    var serverAddr string
    switch sessionData.DataCenter {
    case 1:
        serverAddr = net.JoinHostPort("149.154.175.50", "443")
    case 2:
        serverAddr = net.JoinHostPort("149.154.167.51", "443")
    case 3:
        serverAddr = net.JoinHostPort("149.154.175.100", "443")
    case 4:
        serverAddr = net.JoinHostPort("149.154.167.91", "443")
    case 5:
        serverAddr = net.JoinHostPort("91.108.56.130", "443")
    default:
        serverAddr = net.JoinHostPort("149.154.175.50", "443")
    }
    
    // 计算auth_key_id
    authKeyID := calculateAuthKeyID(sessionData.AuthKey)
    
    // 保存会话数据
    if err := loader.Save(context.Background(), &session.Data{
        DC:        sessionData.DataCenter,
        Addr:      serverAddr,
        AuthKey:   sessionData.AuthKey,
        AuthKeyID: authKeyID,
    }); err != nil {
        return nil, fmt.Errorf("failed to save session data: %w", err)
    }
    
    return storage, nil
}
```

#### 1.4 calculateAuthKeyID - 计算密钥ID
```go
func calculateAuthKeyID(authKey []byte) []byte {
    hash := sha1.Sum(authKey)
    return hash[12:20] // 取SHA1结果的第12-19字节
}
```

#### 1.5 processAuthKey - 处理auth_key数据
```go
func processAuthKey(authKeyData interface{}) ([]byte, error) {
    if authKeyData == nil {
        return nil, fmt.Errorf("auth_key数据为空")
    }
    
    switch data := authKeyData.(type) {
    case []byte:
        return data, nil
    
    case string:
        // 尝试hex解码
        if decoded, err := hex.DecodeString(data); err == nil {
            return decoded, nil
        }
        
        // 尝试base64解码
        if decoded, err := base64.StdEncoding.DecodeString(data); err == nil {
            return decoded, nil
        }
        
        // 直接使用字符串的字节
        return []byte(data), nil
    
    default:
        return nil, fmt.Errorf("不支持的auth_key数据类型: %T", authKeyData)
    }
}
```

#### 1.6 buildSessionQuery - 构建查询语句
```go
func buildSessionQuery(db *sql.DB) (string, error) {
    columns, err := getTableColumns(db, "sessions")
    if err != nil {
        return "", fmt.Errorf("获取sessions表结构失败: %w", err)
    }
    
    hasUserID := hasColumn(columns, "user_id")
    hasIsBot := hasColumn(columns, "is_bot")
    
    var selectFields []string
    selectFields = append(selectFields, "dc_id", "auth_key")
    
    if hasUserID {
        selectFields = append(selectFields, "user_id")
    } else {
        selectFields = append(selectFields, "0 as user_id")
    }
    
    if hasIsBot {
        selectFields = append(selectFields, "is_bot")
    } else {
        selectFields = append(selectFields, "0 as is_bot")
    }
    
    query := fmt.Sprintf("SELECT %s FROM sessions LIMIT 1", strings.Join(selectFields, ", "))
    return query, nil
}
```

#### 1.7 辅助方法
```go
// getTableColumns - 获取表的列名
func getTableColumns(db *sql.DB, tableName string) ([]string, error) {
    rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var columns []string
    for rows.Next() {
        var cid int
        var name, dataType string
        var notNull, pk int
        var defaultValue sql.NullString
        
        if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
            continue
        }
        columns = append(columns, name)
    }
    
    return columns, nil
}

// hasColumn - 检查列是否存在
func hasColumn(columns []string, columnName string) bool {
    for _, col := range columns {
        if strings.EqualFold(col, columnName) {
            return true
        }
    }
    return false
}

// min - 返回较小值
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

---

### 2. TData转SessionString流程

#### 2.1 loadTDataSession - 主入口
```go
func loadTDataSession(tdataPath, phone string) (*SessionData, error) {
    // 使用gotd/td原生支持读取tdata
    accounts, err := tdesktop.Read(tdataPath, nil) // nil表示没有密码
    if err != nil {
        return nil, fmt.Errorf("读取tdata文件夹失败: %w", err)
    }
    
    if len(accounts) == 0 {
        return nil, fmt.Errorf("tdata文件夹中未找到账户信息")
    }
    
    // 使用第一个账户（通常tdata只有一个账户）
    account := accounts[0]
    
    // 使用gotd/td的TDesktopSession转换为标准会话数据
    sessionData, err := session.TDesktopSession(account)
    if err != nil {
        return nil, fmt.Errorf("转换tdata会话格式失败: %w", err)
    }
    
    // 创建内存存储并保存会话数据
    storage := new(session.StorageMemory)
    loader := session.Loader{Storage: storage}
    if err := loader.Save(context.Background(), sessionData); err != nil {
        return nil, fmt.Errorf("保存会话数据失败: %w", err)
    }
    
    // 从storage中获取标准的二进制会话数据
    ctx := context.Background()
    sessionBytes, err := storage.LoadSession(ctx)
    if err != nil {
        return nil, fmt.Errorf("获取会话数据失败: %w", err)
    }
    
    if len(sessionBytes) == 0 {
        return nil, fmt.Errorf("会话数据为空")
    }
    
    // 创建SessionData结构
    result := &SessionData{
        Phone:       phone,
        EncodedData: base64.StdEncoding.EncodeToString(sessionBytes), // Base64编码
        AuthKey:     sessionData.AuthKey,
        DataCenter:  sessionData.DC,
        UserID:      0, // 需要通过验证获取
        Username:    "",
        FirstName:   "",
        LastName:    "",
        IsPremium:   false,
    }
    
    return result, nil
}
```

#### 2.2 isTDataDirectory - 检查tdata文件夹
```go
func isTDataDirectory(path string) bool {
    requiredFiles := []string{"key_datas", "settings0", "maps"}
    
    for _, file := range requiredFiles {
        filePath := filepath.Join(path, file)
        if _, err := os.Stat(filePath); err != nil {
            return false
        }
    }
    
    return true
}
```

---

### 3. 统一入口方法

#### 3.1 loadSessionFromFiles - 自动识别格式
```go
func loadSessionFromFiles(sessionPath, phone string) (*SessionData, error) {
    // 检查是否存在 .session 文件
    sessionFile := filepath.Join(sessionPath, phone+".session")
    if _, err := os.Stat(sessionFile); err == nil {
        return loadPyrogramSession(sessionFile, phone)
    }
    
    // 检查是否存在 tdata 文件夹
    tdataPath := filepath.Join(sessionPath, "tdata")
    if _, err := os.Stat(tdataPath); err == nil {
        return loadTDataSession(tdataPath, phone)
    }
    
    // 检查是否直接是 tdata 格式
    if isTDataDirectory(sessionPath) {
        return loadTDataSession(sessionPath, phone)
    }
    
    return nil, fmt.Errorf("未找到支持的会话文件格式 (.session 或 tdata)")
}
```

---

## 💡 使用示例

### 示例1: Session文件转SessionString
```go
package main

import (
    "fmt"
    "log"
)

func main() {
    sessionFile := "/path/to/user.session"
    phone := "+1234567890"
    
    // 加载session文件
    sessionData, err := loadPyrogramSession(sessionFile, phone)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取SessionString
    sessionString := sessionData.EncodedData
    fmt.Printf("SessionString: %s\n", sessionString)
    fmt.Printf("UserID: %d\n", sessionData.UserID)
    fmt.Printf("DataCenter: %d\n", sessionData.DataCenter)
}
```

### 示例2: TData文件夹转SessionString
```go
package main

import (
    "fmt"
    "log"
)

func main() {
    tdataPath := "/path/to/tdata"
    phone := "+1234567890"
    
    // 加载tdata文件夹
    sessionData, err := loadTDataSession(tdataPath, phone)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取SessionString
    sessionString := sessionData.EncodedData
    fmt.Printf("SessionString: %s\n", sessionString)
    fmt.Printf("DataCenter: %d\n", sessionData.DataCenter)
}
```

### 示例3: 自动识别格式
```go
package main

import (
    "fmt"
    "log"
)

func main() {
    sessionPath := "/path/to/session/folder"
    phone := "+1234567890"
    
    // 自动识别格式并转换
    sessionData, err := loadSessionFromFiles(sessionPath, phone)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取SessionString
    sessionString := sessionData.EncodedData
    fmt.Printf("SessionString: %s\n", sessionString)
}
```

---

## 📌 关键要点

1. **Session文件格式**: `.session` 文件实际上是SQLite数据库，包含 `sessions` 表和可选的 `users` 表
2. **TData格式**: `tdata` 文件夹包含Telegram Desktop的加密数据，需要使用 `tdesktop.Read` 读取
3. **统一输出**: 两种格式最终都转换为 base64 编码的 SessionString，存储在 `EncodedData` 字段
4. **AuthKey验证**: Session格式的auth_key必须是256字节，否则转换会失败
5. **DataCenter映射**: 根据DC ID自动映射到对应的Telegram服务器地址

---

## 🔗 相关文件位置

- 原始实现: `internal/service/session/service.go`
- 行号范围:
  - Session转换: 489-621行
  - TData转换: 623-694行
  - 格式转换: 824-875行
  - 辅助方法: 877-1170行

---

## ✅ 集成检查清单

- [ ] 安装依赖: `github.com/gotd/td` 和 `github.com/mattn/go-sqlite3`
- [ ] 实现 `SessionData` 结构体
- [ ] 实现Session转SessionString的所有方法
- [ ] 实现TData转SessionString的方法
- [ ] 实现辅助方法（hasColumn, getTableColumns等）
- [ ] 测试Session文件转换
- [ ] 测试TData文件夹转换
- [ ] 处理错误情况

---

**注意**: 此文档提取自 `internal/service/session/service.go`，可根据项目需要进行调整和优化。

