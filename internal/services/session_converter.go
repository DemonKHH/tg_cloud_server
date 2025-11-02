package services

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
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

// SessionConverter Session/TData 转换服务
type SessionConverter struct {
	logger *zap.Logger
}

// NewSessionConverter 创建Session转换服务
func NewSessionConverter() *SessionConverter {
	return &SessionConverter{
		logger: logger.Get().Named("session_converter"),
	}
}

// SessionData 会话数据结构
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

// LoadPyrogramSession 加载Pyrogram .session文件并转换为SessionString
func (sc *SessionConverter) LoadPyrogramSession(sessionFile, phone string) (*SessionData, error) {
	// 打开SQLite数据库
	db, err := sql.Open("sqlite3", sessionFile)
	if err != nil {
		return nil, fmt.Errorf("打开session数据库失败: %w", err)
	}
	defer db.Close()

	// 解析session数据
	return sc.parseSessionDatabase(db, phone)
}

// LoadTDataSession 加载TData文件夹并转换为SessionString
func (sc *SessionConverter) LoadTDataSession(tdataPath, phone string) (*SessionData, error) {
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

// LoadSessionFromFiles 从文件加载会话数据（自动识别格式）
func (sc *SessionConverter) LoadSessionFromFiles(sessionPath, phone string) (*SessionData, error) {
	// 检查是否存在 .session 文件
	sessionFile := filepath.Join(sessionPath, phone+".session")
	if _, err := os.Stat(sessionFile); err == nil {
		return sc.LoadPyrogramSession(sessionFile, phone)
	}

	// 检查是否存在 tdata 文件夹
	tdataPath := filepath.Join(sessionPath, "tdata")
	if _, err := os.Stat(tdataPath); err == nil {
		return sc.LoadTDataSession(tdataPath, phone)
	}

	// 检查是否直接是 tdata 格式
	if sc.isTDataDirectory(sessionPath) {
		return sc.LoadTDataSession(sessionPath, phone)
	}

	return nil, fmt.Errorf("未找到支持的会话文件格式 (.session 或 tdata)")
}

// parseSessionDatabase 解析数据库
func (sc *SessionConverter) parseSessionDatabase(db *sql.DB, phone string) (*SessionData, error) {
	sessionData := &SessionData{Phone: phone}

	// 构建查询语句
	query, err := sc.buildSessionQuery(db)
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
	authKey, err = sc.processAuthKey(authKeyData)
	if err != nil {
		return nil, fmt.Errorf("处理auth_key失败: %w", err)
	}

	sessionData.DataCenter = dcID
	sessionData.AuthKey = authKey
	sessionData.UserID = userID

	// 加载用户信息（可选）
	if userID > 0 {
		sc.loadUserInfo(db, userID, sessionData)
	}

	// 转换为gotd格式
	tempSessionData := &SessionData{
		AuthKey:    authKey,
		DataCenter: dcID,
	}

	storage, err := sc.convertPyrogramToGotd(tempSessionData)
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

// convertPyrogramToGotd 格式转换
func (sc *SessionConverter) convertPyrogramToGotd(sessionData *SessionData) (*session.StorageMemory, error) {
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
	authKeyID := sc.calculateAuthKeyID(sessionData.AuthKey)

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

// calculateAuthKeyID 计算密钥ID
func (sc *SessionConverter) calculateAuthKeyID(authKey []byte) []byte {
	hash := sha1.Sum(authKey)
	return hash[12:20] // 取SHA1结果的第12-19字节
}

// processAuthKey 处理auth_key数据
func (sc *SessionConverter) processAuthKey(authKeyData interface{}) ([]byte, error) {
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

// buildSessionQuery 构建查询语句
func (sc *SessionConverter) buildSessionQuery(db *sql.DB) (string, error) {
	columns, err := sc.getTableColumns(db, "sessions")
	if err != nil {
		return "", fmt.Errorf("获取sessions表结构失败: %w", err)
	}

	hasUserID := sc.hasColumn(columns, "user_id")
	hasIsBot := sc.hasColumn(columns, "is_bot")

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

// getTableColumns 获取表的列名
func (sc *SessionConverter) getTableColumns(db *sql.DB, tableName string) ([]string, error) {
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

// hasColumn 检查列是否存在
func (sc *SessionConverter) hasColumn(columns []string, columnName string) bool {
	for _, col := range columns {
		if strings.EqualFold(col, columnName) {
			return true
		}
	}
	return false
}

// loadUserInfo 加载用户信息
func (sc *SessionConverter) loadUserInfo(db *sql.DB, userID int64, sessionData *SessionData) {
	// 检查是否存在users表
	columns, err := sc.getTableColumns(db, "users")
	if err != nil {
		sc.logger.Debug("无法获取users表信息，跳过用户信息加载", zap.Error(err))
		return
	}

	if !sc.hasColumn(columns, "user_id") {
		return
	}

	// 构建查询
	var selectFields []string
	if sc.hasColumn(columns, "username") {
		selectFields = append(selectFields, "username")
	} else {
		selectFields = append(selectFields, "'' as username")
	}
	if sc.hasColumn(columns, "first_name") {
		selectFields = append(selectFields, "first_name")
	} else {
		selectFields = append(selectFields, "'' as first_name")
	}
	if sc.hasColumn(columns, "last_name") {
		selectFields = append(selectFields, "last_name")
	} else {
		selectFields = append(selectFields, "'' as last_name")
	}
	if sc.hasColumn(columns, "is_premium") {
		selectFields = append(selectFields, "is_premium")
	} else {
		selectFields = append(selectFields, "0 as is_premium")
	}

	query := fmt.Sprintf("SELECT %s FROM users WHERE user_id = ? LIMIT 1", strings.Join(selectFields, ", "))

	row := db.QueryRow(query, userID)
	var username, firstName, lastName sql.NullString
	var isPremium bool

	err = row.Scan(&username, &firstName, &lastName, &isPremium)
	if err == nil {
		if username.Valid {
			sessionData.Username = username.String
		}
		if firstName.Valid {
			sessionData.FirstName = firstName.String
		}
		if lastName.Valid {
			sessionData.LastName = lastName.String
		}
		sessionData.IsPremium = isPremium
	}
}

// isTDataDirectory 检查是否为tdata文件夹
func (sc *SessionConverter) isTDataDirectory(path string) bool {
	requiredFiles := []string{"key_datas", "settings0", "maps"}

	for _, file := range requiredFiles {
		filePath := filepath.Join(path, file)
		if _, err := os.Stat(filePath); err != nil {
			return false
		}
	}

	return true
}
