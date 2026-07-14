package xhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Option func(*clientOptions)

type clientOptions struct {
	ctx        context.Context
	timeout    time.Duration
	retryCount int
	middleware []resty.RequestMiddleware
}

func WithMiddleware(m ...resty.RequestMiddleware) Option {
	return func(o *clientOptions) {
		o.middleware = m
	}
}

func WithTimeout(time time.Duration) Option {
	return func(o *clientOptions) {
		o.timeout = time
	}
}

func WithRetryCount(count int) Option {
	return func(o *clientOptions) {
		o.retryCount = count
	}
}

type Client struct {
	opts  clientOptions
	resty *resty.Client
}

const (
	Timeout    = time.Second * 5
	RetryCount = 3
)

type responseRaw struct {
	status     string // e.g. "200 OK"
	statusCode int    // e.g. 200
	proto      string // e.g. "HTTP/1.0"
	time       time.Duration
	result     string
}

// NewXhttpClient 默认不做单例，由应用维护单例的使用
func NewXhttpClient(ctx context.Context, opts ...Option) *Client {
	options := clientOptions{
		ctx:        ctx,
		timeout:    Timeout,
		retryCount: RetryCount,
	}
	for _, o := range opts {
		o(&options)
	}
	return &Client{
		opts: options,
		resty: resty.New().
			SetTimeout(options.timeout).
			SetRetryCount(options.retryCount),
	}
}

func (c *Client) Get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	queryValues := c.SetQueryValues(params)
	resp, err := c.resty.R().
		SetQueryParamsFromValues(queryValues).
		Get(path)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Client) GetResponse(ctx context.Context, path string, params interface{}) (*resty.Response, error) {
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	queryValues := c.SetQueryValues(params)
	resp, err := c.resty.R().
		SetQueryParamsFromValues(queryValues).
		Get(path)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) GetWithHeaders(ctx context.Context, path string, params url.Values, headers map[string]string) ([]byte, error) {
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	queryValues := c.SetQueryValues(params)
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeaders(headers).
		SetQueryParamsFromValues(queryValues).
		Get(path)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Client) PostJson(ctx context.Context, path string, body interface{}) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody(raw).
		Post(path)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("pkg.xhttp.PostJson err. endpoint=[%+v] code=[%+v] resp=[%+v]", path, resp.StatusCode(), resp)
	}
	return resp.Body(), nil
}

func (c *Client) PostWithHeaders(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	resp, err := c.resty.R().
		SetHeaders(headers).
		SetBody(body).
		Post(path)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Client) PostJsonWithHeaders(ctx context.Context, path string, body interface{}, headers map[string]string) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeaders(headers).
		SetBody(raw).
		Post(path)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Client) PostJsonWithParams(ctx context.Context, domain, path string, body interface{}, params interface{}, headers map[string]string) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	for _, m := range c.opts.middleware {
		c.resty.OnBeforeRequest(m)
	}
	queryValues := c.SetQueryValues(params)
	resp, err := c.resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeaders(headers).
		SetQueryParamsFromValues(queryValues).
		SetBody(raw).
		Post(domain + path)
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *Client) SetQueryValues(ifc interface{}) url.Values {
	values := url.Values{}
	if ifc == nil {
		return values
	}
	if mValues, ok := ifc.(url.Values); ok {
		for k := range mValues {
			values.Set(k, mValues.Get(k))
		}
	} else {
		elem := reflect.ValueOf(ifc)
		for i := 0; i < elem.NumField(); i++ {
			field := elem.Field(i)
			kind := field.Kind()
			if (kind == reflect.Ptr || kind == reflect.Array || kind == reflect.Map || kind == reflect.Chan) && field.IsNil() {
				continue
			}
			tag := elem.Type().Field(i).Tag.Get("ArgName")
			if tag == "" {
				tag = elem.Type().Field(i).Name
			}
			arr := strings.Split(tag, ",")
			name := arr[0]
			empty_flag := false //如果值为空 且not_require ，则不放在valuse 里面
			auto_join := false  //如果[]string 且auto_join,则自动按照 ，join成字符串
			if len(arr) >= 2 {
				for i := 1; i < len(arr); i++ {
					if arr[i] == "not_require" {
						empty_flag = true
					}
					if arr[i] == "auto_join" {
						auto_join = true
					}
				}
			}
			switch kind {
			case reflect.Slice:
				is_empty := field.Len() == 0
				if is_empty == false || empty_flag == false {
					var str_arr []string
					for i := 0; i < field.Len(); i++ {
						value, _ := c.getQueryValue(field.Index(i))
						str_arr = append(str_arr, value)
					}
					if auto_join {
						values.Set(name, strings.Join(str_arr, ","))
					} else {
						for _, s := range str_arr {
							values.Add(fmt.Sprintf("%s[]", name), s)
						}
					}
				}
			default:
				value, is_empty := c.getQueryValue(field)
				if is_empty == false || empty_flag == false {
					values.Set(name, value)
				}
			}
		}
	}
	return values
}

func (c *Client) getQueryValue(field reflect.Value) (string, bool) {
	kind := field.Kind()
	var value string
	var is_empty bool
	if (kind == reflect.Ptr || kind == reflect.Array || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Chan) && field.IsNil() {
		return value, is_empty
	}
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := field.Int()
		value = strconv.FormatInt(i, 10)
		is_empty = i == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := field.Uint()
		value = strconv.FormatUint(u, 10)
		is_empty = u == 0
	case reflect.Float32:
		value = strconv.FormatFloat(field.Float(), 'f', 4, 32)
		is_empty = field.Float() == 0
	case reflect.Float64:
		value = strconv.FormatFloat(field.Float(), 'f', 4, 64)
		is_empty = field.Float() == 0
	case reflect.Bool:
		value = strconv.FormatBool(field.Bool())
		is_empty = field.Bool() == false
	case reflect.String:
		value = field.String()
		is_empty = value == ""
	}
	return value, is_empty
}
