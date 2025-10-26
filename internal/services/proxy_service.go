package services

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// ProxyService 代理服务接口
type ProxyService interface {
	CreateProxy(userID uint64, req *models.CreateProxyRequest) (*models.ProxyIP, error)
	GetProxy(userID, proxyID uint64) (*models.ProxyIP, error)
	GetProxies(userID uint64, page, limit int) ([]*models.ProxyIP, int64, error)
	GetProxiesByStatus(userID uint64, status string, page, limit int) ([]*models.ProxyIP, int64, error)
	UpdateProxy(userID, proxyID uint64, req *models.UpdateProxyRequest) (*models.ProxyIP, error)
	DeleteProxy(userID, proxyID uint64) error
	TestProxy(userID, proxyID uint64) (*models.ProxyTestResult, error)
	GetProxyStats(userID uint64) (*models.ProxyStats, error)
}

// proxyService 代理服务实现
type proxyService struct {
	proxyRepo repository.ProxyRepository
	logger    *zap.Logger
}

// NewProxyService 创建代理服务
func NewProxyService(proxyRepo repository.ProxyRepository) ProxyService {
	return &proxyService{
		proxyRepo: proxyRepo,
		logger:    logger.Get().Named("proxy_service"),
	}
}

// CreateProxy 创建代理
func (s *proxyService) CreateProxy(userID uint64, req *models.CreateProxyRequest) (*models.ProxyIP, error) {
	s.logger.Info("Creating proxy",
		zap.Uint64("user_id", userID),
		zap.String("name", req.Name),
		zap.String("host", req.Host))

	proxy := &models.ProxyIP{
		UserID:   userID,
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Protocol: req.Protocol,
		Status:   models.StatusUntested,
	}

	if err := s.proxyRepo.Create(proxy); err != nil {
		s.logger.Error("Failed to create proxy", zap.Error(err))
		return nil, fmt.Errorf("failed to create proxy: %w", err)
	}

	s.logger.Info("Proxy created successfully", zap.Uint64("proxy_id", proxy.ID))
	return proxy, nil
}

// GetProxy 获取代理详情
func (s *proxyService) GetProxy(userID, proxyID uint64) (*models.ProxyIP, error) {
	return s.proxyRepo.GetByUserIDAndID(userID, proxyID)
}

// GetProxies 获取代理列表
func (s *proxyService) GetProxies(userID uint64, page, limit int) ([]*models.ProxyIP, int64, error) {
	offset := (page - 1) * limit
	return s.proxyRepo.GetByUserID(userID, offset, limit)
}

// GetProxiesByStatus 根据状态获取代理列表
func (s *proxyService) GetProxiesByStatus(userID uint64, status string, page, limit int) ([]*models.ProxyIP, int64, error) {
	offset := (page - 1) * limit
	return s.proxyRepo.GetByUserIDAndStatus(userID, status, offset, limit)
}

// UpdateProxy 更新代理
func (s *proxyService) UpdateProxy(userID, proxyID uint64, req *models.UpdateProxyRequest) (*models.ProxyIP, error) {
	proxy, err := s.proxyRepo.GetByUserIDAndID(userID, proxyID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		proxy.Name = req.Name
	}
	if req.Host != "" {
		proxy.Host = req.Host
	}
	if req.Port != 0 {
		proxy.Port = req.Port
	}
	if req.Username != "" {
		proxy.Username = req.Username
	}
	if req.Password != "" {
		proxy.Password = req.Password
	}
	if req.Protocol != "" {
		proxy.Protocol = req.Protocol
	}

	if err := s.proxyRepo.Update(proxy); err != nil {
		s.logger.Error("Failed to update proxy", zap.Error(err))
		return nil, fmt.Errorf("failed to update proxy: %w", err)
	}

	return proxy, nil
}

// DeleteProxy 删除代理
func (s *proxyService) DeleteProxy(userID, proxyID uint64) error {
	// 验证代理所有权
	_, err := s.proxyRepo.GetByUserIDAndID(userID, proxyID)
	if err != nil {
		return err
	}

	return s.proxyRepo.Delete(proxyID)
}

// TestProxy 测试代理连接
func (s *proxyService) TestProxy(userID, proxyID uint64) (*models.ProxyTestResult, error) {
	proxy, err := s.proxyRepo.GetByUserIDAndID(userID, proxyID)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Testing proxy connection",
		zap.Uint64("proxy_id", proxyID),
		zap.String("host", proxy.Host),
		zap.Int("port", proxy.Port))

	result := &models.ProxyTestResult{
		ProxyID:    proxyID,
		Success:    false,
		TestedAt:   time.Now(),
		IPLocation: "",
	}

	// 测试代理连接
	startTime := time.Now()
	err = s.testProxyConnection(proxy)
	duration := time.Since(startTime)

	result.Latency = int(duration.Milliseconds())

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		proxy.Status = models.StatusError
		s.logger.Warn("Proxy test failed",
			zap.Uint64("proxy_id", proxyID),
			zap.Error(err))
	} else {
		result.Success = true
		proxy.Status = models.StatusActive
		s.logger.Info("Proxy test successful",
			zap.Uint64("proxy_id", proxyID),
			zap.Int("latency_ms", result.Latency))
	}

	// 更新代理状态和最后测试时间
	proxy.LastTestAt = &result.TestedAt
	s.proxyRepo.Update(proxy)

	return result, nil
}

// GetProxyStats 获取代理统计信息
func (s *proxyService) GetProxyStats(userID uint64) (*models.ProxyStats, error) {
	return s.proxyRepo.GetStatsByUserID(userID)
}

// testProxyConnection 测试代理连接
func (s *proxyService) testProxyConnection(proxy *models.ProxyIP) error {
	// 构建代理URL
	proxyURL := fmt.Sprintf("%s://%s:%s@%s:%d",
		proxy.Protocol, proxy.Username, proxy.Password, proxy.Host, proxy.Port)

	// 简单的连接测试 - 尝试连接到代理服务器
	address := net.JoinHostPort(proxy.Host, fmt.Sprintf("%d", proxy.Port))
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to proxy server: %w", err)
	}
	defer conn.Close()

	// TODO: 这里可以添加更复杂的代理功能测试
	// 比如通过代理发送HTTP请求到测试URL

	s.logger.Debug("Proxy connection test completed",
		zap.String("proxy_url", proxyURL))

	return nil
}
