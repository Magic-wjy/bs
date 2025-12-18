package httputils

import (
	"bs/constant"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Cli 接口保持不变（注意：Put 建议大写为 PUT，跨包可调用）
type Cli interface {
	GET(urlStr string) (*http.Response, error)
	POST(url string, body interface{}) (*http.Response, error)
	DELETE(urlStr string, params map[string]string) (*http.Response, error)
	Put(url string, body interface{}) (*http.Response, error)
	SetHeader(headers map[string]string) // 设置当前实例的 Header（仅作用于自身）
	ReadResponseBody(resp *http.Response) ([]byte, error)
}

// 核心：单例化 http.Transport（连接池，并发安全）
var (
	singletonTransport *http.Transport
	once               sync.Once // 保证 Transport 仅初始化一次
)

// 初始化单例 Transport（仅执行一次）
func initSingletonTransport() {
	once.Do(func() {
		singletonTransport = &http.Transport{} // 全局唯一连接池，并发安全
	})
}

const (
	DefaultTimeout = time.Second * 120 // 常量名规范化
)

// Client 结构体：每个实例独立持有 Header 和超时，复用单例 Transport
type Client struct {
	cli     *http.Client  // 每个实例独立，但复用单例 Transport
	header  http.Header   // 每个实例独立的 Header，无并发写冲突
	timeout time.Duration // 每个实例独立的超时，避免全局污染
}

// DefaultHTTPClient 创建默认 Client（复用单例 Transport，Header 独立）
func DefaultHTTPClient() Cli {
	initSingletonTransport()
	return &Client{
		cli: &http.Client{
			Transport: singletonTransport, // 复用单例连接池
		},
		header: make(http.Header), // 初始化独立 Header，避免空指针
	}
}

// NewHTTPClient 创建自定义 Client（复用单例 Transport，Header 独立）
func NewHTTPClient(timeout time.Duration, proxy *url.URL) (Cli, error) {
	initSingletonTransport()
	// 仅在首次初始化时配置代理（Transport 单例，避免重复修改）
	if proxy != nil && proxy.Scheme != "" && singletonTransport.Proxy == nil {
		singletonTransport.Proxy = http.ProxyURL(proxy)
	}
	// 处理超时（0 则用默认值）
	if timeout <= constant.Zero {
		timeout = DefaultTimeout
	}
	// 每个 New 都创建新 Client 实例，Header 独立
	return &Client{
		cli: &http.Client{
			Transport: singletonTransport, // 复用单例连接池
			Timeout:   timeout,            // 当前实例独立超时
		},
		header: make(http.Header), // 独立 Header，无锁安全
	}, nil
}

// doRequest 通用请求方法（无锁，Header 仅操作当前实例）
func (c *Client) doRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("JSON 序列化请求体失败：%v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// 创建请求
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%v", err)
	}

	// 复制当前实例的 Header 到请求（仅读当前实例，无并发冲突）
	// 克隆 Header 避免请求修改影响实例本身
	req.Header = c.header.Clone()

	// 默认 Content-Type（仅当前请求生效，不影响实例 Header）
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}

	// 发送请求（移除 defer resp.Body.Close()，调用方负责读取后关闭）
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败：%v", err)
	}

	return resp, nil
}

// SetHeader 设置当前实例的 Header（仅作用于自身，无锁安全）
func (c *Client) SetHeader(headers map[string]string) {
	// 仅修改当前实例的 Header，无并发写同一个资源，无需锁
	for key, value := range headers {
		c.header.Set(key, value)
	}
}

// GET 请求：修复 URL 参数拼接逻辑（更健壮）
func (c *Client) GET(urlStr string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, urlStr, nil)
}

// POST 请求：复用 doRequest
func (c *Client) POST(url string, body interface{}) (*http.Response, error) {
	return c.doRequest(http.MethodPost, url, body)
}

// Put 请求：保持原有命名（建议改为 PUT 规范）
func (c *Client) Put(url string, body interface{}) (*http.Response, error) {
	return c.doRequest(http.MethodPut, url, body)
}

// DELETE 请求：同 GET 优化参数拼接
func (c *Client) DELETE(urlStr string, params map[string]string) (*http.Response, error) {
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		query := values.Encode()
		if query != "" {
			if url.QueryEscape(urlStr) != urlStr {
				urlStr += "&" + query
			} else {
				urlStr += "?" + query
			}
		}
	}
	return c.doRequest(http.MethodDelete, urlStr, nil)
}

// ReadResponseBody 辅助函数：安全读取响应体（调用方必须使用，避免资源泄漏）
func (c *Client) ReadResponseBody(resp *http.Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("响应体为空")
	}
	defer resp.Body.Close() // 读取后关闭，避免连接泄漏
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败：%v", err)
	}
	return body, nil
}
