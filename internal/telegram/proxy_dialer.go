package telegram

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// createProxyDialer 创建代理拨号器
func createProxyDialer(config *ProxyConfig) (proxy.Dialer, error) {
	switch config.Protocol {
	case "http", "https":
		return createHTTPProxyDialer(config)
	case "socks5":
		return createSOCKS5ProxyDialer(config)
	default:
		return nil, fmt.Errorf("unsupported proxy protocol: %s", config.Protocol)
	}
}

// createHTTPProxyDialer 创建HTTP代理拨号器
func createHTTPProxyDialer(config *ProxyConfig) (proxy.Dialer, error) {
	proxyURL := &url.URL{
		Scheme: config.Protocol,
		Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
	}

	if config.Username != "" && config.Password != "" {
		proxyURL.User = url.UserPassword(config.Username, config.Password)
	}

	// 创建HTTP传输层
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// 包装为proxy.Dialer接口
	return &httpProxyDialer{
		transport: transport,
		proxyURL:  proxyURL,
	}, nil
}

// createSOCKS5ProxyDialer 创建SOCKS5代理拨号器
func createSOCKS5ProxyDialer(config *ProxyConfig) (proxy.Dialer, error) {
	proxyAddr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var auth *proxy.Auth
	if config.Username != "" && config.Password != "" {
		auth = &proxy.Auth{
			User:     config.Username,
			Password: config.Password,
		}
	}

	return proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
}

// httpProxyDialer HTTP代理拨号器实现
type httpProxyDialer struct {
	transport *http.Transport
	proxyURL  *url.URL
}

// Dial 实现proxy.Dialer接口
func (d *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	// 对于HTTP代理，我们需要建立到代理服务器的连接
	// 然后通过CONNECT方法建立隧道
	conn, err := net.DialTimeout("tcp", d.proxyURL.Host, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %w", err)
	}

	// 发送CONNECT请求
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)

	// 添加代理认证（如果需要）
	if d.proxyURL.User != nil {
		username := d.proxyURL.User.Username()
		password, _ := d.proxyURL.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}

	connectReq += "\r\n"

	if _, err := conn.Write([]byte(connectReq)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
	}

	// 读取响应
	response := make([]byte, 1024)
	if _, err := conn.Read(response); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	// 简单检查响应状态（实际实现应该更完整）
	if !contains(response, []byte("200 Connection established")) &&
		!contains(response, []byte("200 OK")) {
		conn.Close()
		return nil, fmt.Errorf("proxy connection failed: %s", string(response))
	}

	return conn, nil
}

// testProxyConnection 测试代理连接
func testProxyConnection(config *ProxyConfig) error {
	dialer, err := createProxyDialer(config)
	if err != nil {
		return fmt.Errorf("failed to create proxy dialer: %w", err)
	}

	// 尝试连接到Telegram服务器
	conn, err := dialer.Dial("tcp", "149.154.167.50:443") // Telegram DC1
	if err != nil {
		return fmt.Errorf("failed to connect through proxy: %w", err)
	}
	defer conn.Close()

	// 设置连接超时
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("failed to set connection deadline: %w", err)
	}

	return nil
}

// contains 检查字节数组是否包含子数组
func contains(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		found := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

// proxyDialerAdapter 将proxy.Dialer适配为net.Dialer的DialContext函数
type proxyDialerAdapter struct {
	dialer proxy.Dialer
}

// DialContext 实现context-aware dialer，供gotd/td使用
func (p *proxyDialerAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// proxy.Dialer接口不支持context，但我们可以通过超时控制来实现类似功能
	type result struct {
		conn net.Conn
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		conn, err := p.dialer.Dial(network, addr)
		resultChan <- result{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}
		// 设置连接的deadline（如果context有deadline）
		if deadline, ok := ctx.Deadline(); ok {
			if err := res.conn.SetDeadline(deadline); err != nil {
				res.conn.Close()
				return nil, fmt.Errorf("failed to set connection deadline: %w", err)
			}
		}
		return res.conn, nil
	}
}
