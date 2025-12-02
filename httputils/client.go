package httputils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HttpCli interface {
	GET(ctx context.Context, urlStr string, params map[string]string, headers map[string]string) (*http.Response, error)
	POST(ctx context.Context, url string, body interface{}, headers map[string]string) (*http.Response, error)
	DELETE(ctx context.Context, urlStr string, params map[string]string, headers map[string]string) (*http.Response, error)
}

var httpClient = &Client{}

const (
	TIMEOUT = time.Second * 120
)

type Client struct {
	cli     *http.Client
	timeout time.Duration // 默认超时时间
	proxy   string
}

// NewHTTPClient 创建 HTTP 客户端实例
func NewHTTPClient(timeout time.Duration, proxy string) error {
	transport := &http.Transport{}
	// 配置代理（如果传入了代理地址）
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			fmt.Printf("警告：代理配置无效，将忽略代理：%v\n", err)
			return err
		}
	}
	if timeout == 0 {
		timeout = TIMEOUT
	}
	httpClient = &Client{
		cli: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		timeout: timeout,
		proxy:   proxy,
	}
	return nil
}

// 通用请求方法（内部使用，封装重复逻辑）
func (c *Client) doRequest(ctx context.Context, method, url string, headers map[string]string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	// 处理请求体（支持 JSON 序列化）
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("JSON 序列化请求体失败：%v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%v", err)
	}

	// 设置默认请求头（JSON 格式）
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	// 覆盖/添加自定义请求头
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 发送请求
	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败：%v", err)
	}
	defer resp.Body.Close() // 确保响应体关闭
	return resp, nil
}

// GET 请求：支持 URL 参数、自定义请求头
func (c *Client) GET(ctx context.Context, urlStr string, params map[string]string, headers map[string]string) (*http.Response, error) {
	// 拼接 URL 参数
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		if urlQuery := values.Encode(); urlQuery != "" {
			urlStr += "?" + urlQuery
		}
	}
	// 发送 GET 请求（无请求体）
	return c.doRequest(ctx, http.MethodGet, urlStr, headers, nil)
}

// POST 请求：支持 JSON 请求体、自定义请求头
// url: 请求地址
// body: 请求体（可序列化的结构体/Map）
// headers: 自定义请求头（可选，传 nil 用默认）
// respObj: 响应数据反序列化的目标对象（如 &Result{}）
func (c *Client) POST(ctx context.Context, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPost, url, headers, body)
}

// Put 请求：用法同 Post
func (c *Client) Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.doRequest(ctx, http.MethodPut, url, headers, body)
}

// DELETE 请求：支持 URL 参数、自定义请求头
func (c *Client) DELETE(ctx context.Context, urlStr string, params map[string]string, headers map[string]string) (*http.Response, error) {
	// 拼接 URL 参数
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		if urlQuery := values.Encode(); urlQuery != "" {
			urlStr += "?" + urlQuery
		}
	}
	return c.doRequest(ctx, http.MethodDelete, urlStr, headers, nil)
}
