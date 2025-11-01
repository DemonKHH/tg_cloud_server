package services

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

// AccountParser 账号文件解析服务
type AccountParser struct {
	logger *zap.Logger
}

// NewAccountParser 创建账号解析服务
func NewAccountParser() *AccountParser {
	return &AccountParser{
		logger: logger.Get().Named("account_parser"),
	}
}

// ParsedAccount 解析后的账号信息
type ParsedAccount struct {
	Phone       string
	SessionData string
	Error       string
	Source      string // 标识来源文件
}

// ParseAccountFiles 解析账号文件（支持zip、单个文件、文件夹）
func (p *AccountParser) ParseAccountFiles(filePath string) ([]*ParsedAccount, error) {
	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件不存在: %v", err)
	}

	var accounts []*ParsedAccount

	if info.IsDir() {
		// 如果是目录，递归处理
		accounts, err = p.parseDirectory(filePath)
	} else if strings.HasSuffix(strings.ToLower(filePath), ".zip") {
		// 如果是zip文件，先解压
		accounts, err = p.parseZipFile(filePath)
	} else {
		// 单个文件处理
		accounts, err = p.parseSingleFile(filePath)
	}

	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("未能从文件中解析出账号信息")
	}

	return accounts, nil
}

// parseZipFile 解析zip文件
func (p *AccountParser) parseZipFile(zipPath string) ([]*ParsedAccount, error) {
	p.logger.Info("开始解析zip文件", zap.String("path", zipPath))

	// 创建临时解压目录
	tempDir, err := os.MkdirTemp("", "account_parse_*")
	if err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 打开zip文件
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("打开zip文件失败: %v", err)
	}
	defer r.Close()

	// 解压所有文件
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// 创建目标文件路径
		targetPath := filepath.Join(tempDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			p.logger.Warn("创建目录失败", zap.String("path", targetPath), zap.Error(err))
			continue
		}

		// 解压文件
		rc, err := f.Open()
		if err != nil {
			p.logger.Warn("打开zip内文件失败", zap.String("file", f.Name), zap.Error(err))
			continue
		}

		dst, err := os.Create(targetPath)
		if err != nil {
			rc.Close()
			p.logger.Warn("创建目标文件失败", zap.String("path", targetPath), zap.Error(err))
			continue
		}

		_, err = io.Copy(dst, rc)
		dst.Close()
		rc.Close()

		if err != nil {
			p.logger.Warn("解压文件失败", zap.String("file", f.Name), zap.Error(err))
			continue
		}
	}

	// 解析解压后的目录
	return p.parseDirectory(tempDir)
}

// parseDirectory 解析目录
func (p *AccountParser) parseDirectory(dirPath string) ([]*ParsedAccount, error) {
	p.logger.Info("开始解析目录", zap.String("path", dirPath))

	var accounts []*ParsedAccount

	// 遍历目录
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			// 检查是否是tdata目录
			if info.Name() == "tdata" {
				account, err := p.parseTDataFolder(path)
				if err != nil {
					p.logger.Warn("解析tdata失败", zap.String("path", path), zap.Error(err))
					return nil
				}
				if account != nil {
					accounts = append(accounts, account)
				}
			}
			return nil
		}

		// 处理文件
		lowerName := strings.ToLower(info.Name())
		if strings.HasSuffix(lowerName, ".session") {
			account, err := p.parseSessionFile(path)
			if err != nil {
				p.logger.Warn("解析session文件失败", zap.String("path", path), zap.Error(err))
				return nil
			}
			if account != nil {
				accounts = append(accounts, account)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %v", err)
	}

	return accounts, nil
}

// parseSingleFile 解析单个文件
func (p *AccountParser) parseSingleFile(filePath string) ([]*ParsedAccount, error) {
	p.logger.Info("开始解析单个文件", zap.String("path", filePath))

	fileName := strings.ToLower(filepath.Base(filePath))
	var account *ParsedAccount
	var err error

	if strings.HasSuffix(fileName, ".session") {
		account, err = p.parseSessionFile(filePath)
	} else if filepath.Dir(filePath) != "." && strings.Contains(filepath.Base(filepath.Dir(filePath)), "tdata") {
		// 可能是tdata相关的文件
		account, err = p.parseTDataFolder(filepath.Dir(filePath))
	} else {
		// 尝试作为session文件解析
		account, err = p.parseSessionFile(filePath)
	}

	if err != nil {
		return nil, err
	}

	if account == nil {
		return nil, fmt.Errorf("未能解析出账号信息")
	}

	return []*ParsedAccount{account}, nil
}

// parseSessionFile 解析.session文件（gotd/td格式）
func (p *AccountParser) parseSessionFile(filePath string) (*ParsedAccount, error) {
	p.logger.Debug("解析session文件", zap.String("path", filePath))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("文件为空")
	}

	// gotd/td的session数据通常是二进制，我们直接转换为base64字符串存储
	// 或者如果是JSON格式，则尝试解析
	sessionString := ""

	// 尝试解析为JSON格式（某些工具导出的session可能是JSON）
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		// 如果是JSON格式，提取session字段
		if sessionStr, ok := jsonData["session"].(string); ok {
			sessionString = sessionStr
		} else if sessionBytes, ok := jsonData["session"].([]byte); ok {
			sessionString = base64.StdEncoding.EncodeToString(sessionBytes)
		} else {
			// 整个JSON作为session数据
			jsonBytes, _ := json.Marshal(jsonData)
			sessionString = base64.StdEncoding.EncodeToString(jsonBytes)
		}
	} else {
		// 二进制数据，转换为base64
		sessionString = base64.StdEncoding.EncodeToString(data)
	}

	// 尝试从文件名或文件内容中提取手机号
	phone := p.extractPhoneFromPath(filePath)
	if phone == "" {
		// 尝试从session数据中提取手机号
		if jsonData != nil {
			if p, ok := jsonData["phone"].(string); ok {
				phone = p
			}
		}
	}

	if phone == "" {
		phone = "unknown" // 如果无法提取，使用占位符
	}

	return &ParsedAccount{
		Phone:       phone,
		SessionData: sessionString,
		Source:      filepath.Base(filePath),
	}, nil
}

// parseTDataFolder 解析tdata文件夹（Telegram Desktop格式）
func (p *AccountParser) parseTDataFolder(tdataPath string) (*ParsedAccount, error) {
	p.logger.Debug("解析tdata文件夹", zap.String("path", tdataPath))

	// Telegram Desktop的tdata结构通常是：
	// tdata/
	//   - key_data (可选)
	//   - D877F783D5D3EF8C/ (账户ID目录，16位十六进制)
	//     - key_datas
	//     - auth

	var allFiles []string

	// 收集tdata目录下的所有文件（用于转换为session）
	err := filepath.Walk(tdataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			allFiles = append(allFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历tdata目录失败: %v", err)
	}

	if len(allFiles) == 0 {
		return nil, fmt.Errorf("tdata目录为空")
	}

	// 将整个tdata文件夹打包为session数据
	// 创建一个内存buffer来存储所有文件的数据
	var sessionData bytes.Buffer

	// 读取关键文件：key_data, key_datas, auth等
	keyFiles := []string{"key_data", "key_datas", "auth"}
	for _, keyFile := range keyFiles {
		filePath := filepath.Join(tdataPath, keyFile)
		if data, err := os.ReadFile(filePath); err == nil {
			sessionData.Write(data)
		}

		// 也在子目录中查找
		entries, _ := os.ReadDir(tdataPath)
		for _, entry := range entries {
			if entry.IsDir() && len(entry.Name()) == 16 {
				subFilePath := filepath.Join(tdataPath, entry.Name(), keyFile)
				if data, err := os.ReadFile(subFilePath); err == nil {
					sessionData.Write(data)
				}
			}
		}
	}

	// 如果关键文件都没有，读取所有文件（最多前几个文件，避免太大）
	if sessionData.Len() == 0 && len(allFiles) > 0 {
		maxFiles := 10
		if len(allFiles) < maxFiles {
			maxFiles = len(allFiles)
		}
		for i := 0; i < maxFiles; i++ {
			if data, err := os.ReadFile(allFiles[i]); err == nil && len(data) < 1024*1024 { // 限制单个文件1MB
				sessionData.Write(data)
			}
		}
	}

	if sessionData.Len() == 0 {
		return nil, fmt.Errorf("tdata目录中未找到有效的session数据")
	}

	// 将tdata转换为base64字符串
	sessionString := base64.StdEncoding.EncodeToString(sessionData.Bytes())

	// 尝试从路径或文件中提取手机号
	phone := p.extractPhoneFromPath(tdataPath)
	if phone == "" {
		phone = "unknown"
	}

	return &ParsedAccount{
		Phone:       phone,
		SessionData: sessionString,
		Source:      "tdata",
	}, nil
}

// extractPhoneFromPath 从文件路径中提取手机号
func (p *AccountParser) extractPhoneFromPath(path string) string {
	// 尝试从文件名或路径中提取手机号
	// 例如：+1234567890.session, 1234567890.session, account_+1234567890.session

	// 获取文件名（不含扩展名）
	baseName := filepath.Base(path)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// 查找手机号模式：+数字 或 纯数字
	parts := strings.FieldsFunc(baseName, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		// 检查是否是手机号格式
		if strings.HasPrefix(part, "+") {
			// +开头的号码
			if len(part) > 1 && isDigits(part[1:]) {
				return part
			}
		} else if isDigits(part) && len(part) >= 10 {
			// 纯数字，长度至少10位
			return "+" + part
		}
	}

	return ""
}

// isDigits 检查字符串是否全为数字
func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
