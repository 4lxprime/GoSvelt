package gosvelt

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

func (c *Context) CSet(key, value string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.store == nil {
		c.store = make(Map)
	}

	if pool, ok := c.gosvelt.storePool.Get().(Map); ok {
		pool[key] = value
		c.store = pool

	} else {
		c.store[key] = value
	}
}

func (c *Context) CGet(key string) string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.store == nil {
		return ""
	}

	if pool, ok := c.gosvelt.storePool.Get().(Map); ok {
		value, ok := pool[key]
		if ok {
			return value.(string)
		}

		value = c.store[key]
		pool[key] = value
		c.store = pool

		return value.(string)
	}

	if value, ok := c.store[key]; ok {
		return value.(string)
	}

	return ""
}

func (c *Context) CDel(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.store == nil {
		return
	}

	if pool, ok := c.gosvelt.storePool.Get().(Map); ok {
		delete(pool, key)
		c.store = pool

	} else {
		delete(c.store, key)
	}
}

func (c *Context) CReset() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.store = make(Map)
	c.gosvelt.storePool.Put(c.store)
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
		// create a map of placeholders to values
		placeholders := make(map[string]string)
		for i, arg := range args[1:] {
			placeholders[fmt.Sprintf("%d", i+1)] = fmt.Sprint(arg)
		}

		// replace placeholders in the template with values
		for placeholder, value := range placeholders {
			t = strings.ReplaceAll(t, fmt.Sprintf("&{%s}", placeholder), value)
		}

		output = t

	case Map:
		// create a map of placeholders to values
		placeholders := make(map[string]string)
		for key, value := range args[0].(Map) {
			placeholders[fmt.Sprintf("&{%s}", key)] = fmt.Sprint(value)
		}

		// replace placeholders in the template with values
		for placeholder, value := range placeholders {
			t = strings.ReplaceAll(t, placeholder, value)
		}

		output = t

	default:
		return fmt.Errorf("args must be...string or gosvelt.Map")
	}

	c.SetCType(MTextHtmlUTF8)

	c.SetStatusCode(code)
	c.Write([]byte(output))

	return nil
}

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
	return fmt.Sprintf("%s", c.fasthttpCtx.UserValue(key))
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
	c.Res().AppendBody(body) // todo: remplace fasthttp context by something else
}

// set response header
func (c *Context) SetHeader(key, value string) {
	c.Res().Header.Set(key, value)
}

// set response status code in int
func (c *Context) SetStatusCode(code int) {
	c.Res().Header.SetStatusCode(code)
}

// set response content type
func (c *Context) SetCType(ctype string) {
	c.Res().Header.Set("Content-Type", ctype)
}
