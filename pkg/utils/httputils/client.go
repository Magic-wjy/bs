package httputils

import (
	"net/http"
	"time"
)

// Cli 接口保持不变（注意：Put 建议大写为 PUT，跨包可调用）
type Cli interface {
	GET(urlStr string) (*http.Response, error)
	POST(url string, body interface{}) (*http.Response, error)
	DELETE(urlStr string, params map[string]string) (*http.Response, error)
	PUT(url string, body interface{}) (*http.Response, error)
	SetHeader(headers map[string]string) // 设置当前实例的 Header（仅作用于自身）
	ReadResponseBody(resp *http.Response) ([]byte, error)
}

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
