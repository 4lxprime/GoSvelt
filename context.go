package gosvelt

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
)

type Context struct {
	gosvelt     *GoSvelt
	fasthttpCtx *fasthttp.RequestCtx
	Ctx         context.Context
	store       Map
	lock        sync.RWMutex
}

func (gs *GoSvelt) newContext() any {
	return &Context{
		gosvelt: gs,
		Ctx:     context.Background(),
		lock:    sync.RWMutex{},
	}
}

func (c *Context) update(ctx *fasthttp.RequestCtx) {
	c.Ctx = context.Background()
	c.fasthttpCtx = ctx
	c.store = make(Map)
}

func (c *Context) reset() {
	c.fasthttpCtx = nil
	c.store = nil
}

func (c *Context) Req() *fasthttp.Request {
	return &c.fasthttpCtx.Request
}

func (c *Context) Res() *fasthttp.Response {
	return &c.fasthttpCtx.Response
}

func (c *Context) Path() string {
	return string(c.fasthttpCtx.URI().Path())
}

func (c *Context) Method() string {
	return string(c.fasthttpCtx.Request.Header.Method())
}

func (c *Context) Args() *fasthttp.Args {
	return c.fasthttpCtx.QueryArgs()
}

// CONTEXT STORE -->

func (c *Context) Set(key, value string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.store == nil {
		c.store = make(Map)
	}

	c.store[key] = value
}

func (c *Context) Get(key string) string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.store[key].(string)
}

func (c *Context) CacheReset() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.store = make(Map)
}

// CONTEXT RESPONSES -->

func (c *Context) Html(code int, t string, args ...any) error {
	// check if it's a file or an string
	// and if it's a path read it
	if !(filepath.Ext(t) == "") {
		file, err := os.Open(t)
		if err != nil {
			return err
		}
		defer file.Close()

		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		t = string(content)
	}

	var output string

	switch args[0].(type) {
	case string:
		// find pattern &{x} and replace it with an given arg
		// where x is an string
		re := regexp.MustCompile(`&{(\d+)}`)
		output = re.ReplaceAllStringFunc(t, func(match string) string {
			digitStr := re.FindStringSubmatch(match)[1]
			digit, _ := strconv.Atoi(digitStr)

			if digit >= 1 && digit <= len(args) {
				// convert []any into []string
				strArgs := make([]string, len(args))
				for i, v := range args {
					strArgs[i] = v.(string)
				}
				return strArgs[digit-1]
			}

			return match
		})

	case Map:
		// find pattern &{x} and replace it with an given arg
		// where x is an string without ""
		re := regexp.MustCompile(`&{(\w+)}`)
		output = re.ReplaceAllStringFunc(t, func(match string) string {
			key := re.FindStringSubmatch(match)[1]

			if value, ok := args[0].(Map)[key]; ok {
				if strValue, ok := value.(string); ok {
					return strValue
				}
			}

			return match
		})

	default:
		return fmt.Errorf("args must be ...string or gosvelt.Map")
	}

	c.SetCType("text/html; charset=UTF-8")

	c.SetStatusCode(code)
	c.Write([]byte(output))

	return nil
}

// todo: fix "Error occurred: not found" but it work
func (c *Context) File(code int, file string, compress ...bool) error {
	fs := &fasthttp.FS{
		Root:               "",
		AllowEmptyRoot:     true,
		GenerateIndexPages: false,
		Compress:           true,
		CacheDuration:      10 * time.Second,
		IndexNames:         []string{"index.html"},
	}
	handler := fs.NewRequestHandler()

	if len(compress) == 0 || !compress[0] {
		c.fasthttpCtx.Request.Header.Del("Accept-Encoding")
	}

	// if file is absolute
	if len(file) == 0 || !filepath.IsAbs(file) {
		hasTrailingSlash := len(file) > 0 && (file[len(file)-1] == '/' || file[len(file)-1] == '\\')

		var err error
		file = filepath.FromSlash(file)
		if file, err = filepath.Abs(file); err != nil {
			return fmt.Errorf("failed to determine abs file path: %w", err)
		}
		if hasTrailingSlash {
			file += "/"
		}
	}

	file = filepath.FromSlash(file)
	burl := c.Path()

	defer c.Req().SetRequestURI(burl)
	c.Req().SetRequestURI(file)

	s := c.Res().StatusCode()

	handler(c.fasthttpCtx)

	fss := c.Res().StatusCode()

	if s != fss || fss == 404 {
		return fmt.Errorf("fileserver: file %s not found", file)
	}

	return nil
}

// return ws connection
// NOTE: this need websocket.FastHTTPHandler handler
// and all ws code will be in the arg handler
func (c *Context) Ws(handler websocket.FastHTTPHandler) error {
	upgrader := websocket.FastHTTPUpgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	err := upgrader.Upgrade(c.fasthttpCtx, handler)
	if err != nil {
		return err
	}

	return nil
}

// return json datas to client
func (c *Context) Json(code int, j interface{}) error {
	c.SetCType(MAppJsonUTF8)

	jsonData, err := json.Marshal(j)
	if err != nil {
		return err
	}

	c.SetStatusCode(code)
	c.Write(jsonData)

	return nil
}

// return proto datas to client
func (c *Context) Proto(code int, p proto.Message) error {
	c.SetCType(MOctStream)

	dataBytes, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	c.SetStatusCode(code)
	c.Write(dataBytes)

	return nil
}

// return text datas to client
func (c *Context) Text(code int, t string) error {
	c.SetCType(MTextPlainUTF8)

	c.SetStatusCode(code)
	c.Write([]byte(t))

	return nil
}

// return bytes datas to client
func (c *Context) Blob(code int, b []byte) error {
	c.SetCType(MOctStream)

	c.SetStatusCode(code)
	c.Write(b)

	return nil
}

func (c *Context) Redirect(code int, url string) error {
	if code > 308 || code < 300 {
		return fmt.Errorf("redirect: code must be between 300 and 308")
	}

	// TODO: add redirect
	c.fasthttpCtx.Redirect(url, code)

	return nil
}

// CONTEXT DATAS -->

// get the page cookies with key
func (c *Context) Cookie(key string) string {
	return string(c.Req().Header.Cookie(key))
}

// get the protocol of the request
func (c *Context) Protocol() string {
	if c.fasthttpCtx.IsTLS() {
		return "https"
	}

	return "http"
}

// true if request is secure by https
func (c *Context) Secure() bool {
	return c.Protocol() == "https"
}

// get the url params with key
func (c *Context) Param(key string) string {
	return fmt.Sprintf("%s", c.fasthttpCtx.UserValue(key)) // todo: remplace fasthttp context by something else
}

// add an cookie
func (c *Context) SetCookie(k, v string, expire time.Time) {
	cookie := &fasthttp.Cookie{}

	cookie.SetKey(k)
	cookie.SetValue(v)
	cookie.SetExpire(expire)

	c.Res().Header.SetCookie(cookie)
}

// CONTEXT WRITERS -->

// write to client
func (c *Context) Write(body []byte) {
	c.fasthttpCtx.Write(body) // todo: remplace fasthttp context by something else
}

// set response header
func (c *Context) SetHeader(key, value string) {
	c.Res().Header.Set(key, value)
}

// set response status code in int
func (c *Context) SetStatusCode(code int) {
	c.Res().SetStatusCode(code)
}

// set response content type
func (c *Context) SetCType(ctype string) {
	c.SetHeader("Content-Type", ctype)
}
