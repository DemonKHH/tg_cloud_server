package services

import (
	"archive/zip"
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
	logger           *zap.Logger
	sessionConverter *SessionConverter
}

// NewAccountParser 创建账号解析服务
func NewAccountParser() *AccountParser {
	return &AccountParser{
		logger:           logger.Get().Named("account_parser"),
		sessionConverter: NewSessionConverter(),
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

// parseSessionFile 解析.session文件（Pyrogram格式）
func (p *AccountParser) parseSessionFile(filePath string) (*ParsedAccount, error) {
	p.logger.Debug("解析session文件", zap.String("path", filePath))

	// 尝试从文件名中提取手机号
	phone := p.extractPhoneFromPath(filePath)
	if phone == "" {
		phone = "unknown"
	}

	// 使用SessionConverter转换.session文件
	sessionData, err := p.sessionConverter.LoadPyrogramSession(filePath, phone)
	if err != nil {
		p.logger.Warn("使用Pyrogram格式解析失败，尝试其他格式", zap.String("path", filePath), zap.Error(err))

		// 如果转换失败，可能是gotd格式的session文件，尝试直接读取
		data, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return nil, fmt.Errorf("读取文件失败: %v (原始错误: %w)", readErr, err)
		}

		if len(data) == 0 {
			return nil, fmt.Errorf("文件为空")
		}

		// 尝试解析为JSON格式（某些工具导出的session可能是JSON）
		var jsonData map[string]interface{}
		var sessionString string
		if jsonErr := json.Unmarshal(data, &jsonData); jsonErr == nil {
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

			// 尝试从JSON中提取手机号
			if phone == "unknown" && jsonData != nil {
				if p, ok := jsonData["phone"].(string); ok {
					phone = p
				}
			}
		} else {
			// 二进制数据，转换为base64（gotd格式）
			sessionString = base64.StdEncoding.EncodeToString(data)
		}

		return &ParsedAccount{
			Phone:       phone,
			SessionData: sessionString,
			Source:      filepath.Base(filePath),
		}, nil
	}

	// 成功转换，使用转换后的数据
	return &ParsedAccount{
		Phone:       sessionData.Phone,
		SessionData: sessionData.EncodedData,
		Source:      filepath.Base(filePath),
	}, nil
}

// parseTDataFolder 解析tdata文件夹（Telegram Desktop格式）
func (p *AccountParser) parseTDataFolder(tdataPath string) (*ParsedAccount, error) {
	p.logger.Debug("解析tdata文件夹", zap.String("path", tdataPath))

	// 尝试从路径中提取手机号
	phone := p.extractPhoneFromPath(tdataPath)
	if phone == "" {
		phone = "unknown"
	}

	// 使用SessionConverter转换tdata文件夹
	sessionData, err := p.sessionConverter.LoadTDataSession(tdataPath, phone)
	if err != nil {
		p.logger.Warn("使用TData转换器解析失败", zap.String("path", tdataPath), zap.Error(err))
		return nil, fmt.Errorf("解析tdata文件夹失败: %w", err)
	}

	// 成功转换，使用转换后的数据
	return &ParsedAccount{
		Phone:       sessionData.Phone,
		SessionData: sessionData.EncodedData,
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
