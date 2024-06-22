package gosvelt

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/buaazp/fasthttprouter"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
)

type Map map[string]interface{}

func (m Map) Add(key string, value interface{}) {
	m[key] = value
}

func (m Map) Del(key string) {
	delete(m, key)
}

func (m Map) Get(key string) interface{} {
	return m[key]
}

const (
	CharsetUTF8 = "charset=UTF-8"

	// Methods
	MGet     = http.MethodGet     // get
	MPost    = http.MethodPost    // post
	MPut     = http.MethodPut     // put
	MDelete  = http.MethodDelete  // delete
	MConnect = http.MethodConnect // connect
	MOptions = http.MethodOptions // options

	// Mime
	MAppJSON       = "application/json"                  // json
	MAppProto      = "application/protobuf"              // protobuf
	MAppJS         = "application/javascript"            // js
	MAppXML        = "application/xml"                   // xml
	MAppForm       = "application/x-www-form-urlencoded" // form
	MOctStream     = "application/octet-stream"          // octet stream
	MTextPlain     = "text/plain"                        // text
	MTextHTML      = "text/html"                         // html
	MTextXML       = "text/xml"                          // xml text
	MAppJsonUTF8   = MAppJSON + "; " + CharsetUTF8       // json utf8
	MAppJsUTF8     = MAppJS + "; " + CharsetUTF8         // js utf8
	MAppXmlUTF8    = MAppXML + "; " + CharsetUTF8        // xml utf8
	MTextPlainUTF8 = MTextPlain + "; " + CharsetUTF8     // text utf8
	MTextHtmlUTF8  = MTextHTML + "; " + CharsetUTF8      // html utf8
	MTextXmlUTF8   = MTextXML + "; " + CharsetUTF8       // xml text utf8
)

type Options struct {
	log            bool
	http2          bool
	errorHandler   ErrorHandlerFunc
	tailwindcssCfg *string
	postcssCfg     *string
}
type Option func(*Options)

type GoSvelt struct {
	config            *Options
	server            *fasthttp.Server
	router            *fasthttprouter.Router
	pool              sync.Pool
	storePool         sync.Pool
	middlewares       map[string]MiddlewareFunc
	svelteMiddlewares map[string]SvelteMiddlewareFunc
	errHandler        ErrorHandlerFunc
}

var (
	WithLog = func(o *Options) {
		o.log = true
	}
	WithHttp2 = func(o *Options) {
		o.http2 = true
	}
	WithErrorHandler = func(errorHandler ErrorHandlerFunc) Option {
		return func(o *Options) {
			o.errorHandler = errorHandler
		}
	}
	WithTailwind = func(tailwindConfig string) Option {
		return func(o *Options) {
			o.tailwindcssCfg = &tailwindConfig
		}
	}
	WithPostcss = func(postcssConfig string) Option {
		return func(o *Options) {
			o.postcssCfg = &postcssConfig
		}
	}
)

func initOptions(options []Option) *Options {
	opts := &Options{
		log:            false,
		http2:          false,
		errorHandler:   defaultErrorHandler,
		tailwindcssCfg: nil,
		postcssCfg:     nil,
	}

	for _, opt := range options {
		opt(opts)
	}

	return opts
}

func New(options ...Option) *GoSvelt {
	opts := initOptions(options)

	gs := &GoSvelt{
		config:            opts,
		server:            &fasthttp.Server{},
		router:            fasthttprouter.New(),
		pool:              sync.Pool{},
		storePool:         sync.Pool{},
		middlewares:       make(map[string]MiddlewareFunc),
		svelteMiddlewares: make(map[string]SvelteMiddlewareFunc),
	}

	gs.router.NotFound = func(ctx *fasthttp.RequestCtx) {
		err := fmt.Errorf(
			http.StatusText(http.StatusNotFound),
		)

		opts.errorHandler(ctx, err)
	}
	gs.pool.New = gs.newContext
	gs.storePool.New = func() interface{} { return make(Map) }

	return gs
}

func (gs *GoSvelt) Start(addr string) {
	gs.server.Handler = gs.router.Handler

	if gs.config.http2 {
		http2.ConfigureServer(gs.server, http2.ServerConfig{})
	}

	if _, err := os.Stat(svelteWorkdir); os.IsExist(err) {
		err = cleanDir(svelteWorkdir)
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("GoSvelt is started on [:%s]\n", addr)

	if err := gs.server.ListenAndServe(addr); err != nil {
		panic(err)
	}
}

func (gs *GoSvelt) StartTLS(addr, cert, key string) {
	gs.server.Handler = gs.router.Handler

	if gs.config.http2 {
		http2.ConfigureServer(gs.server, http2.ServerConfig{})
	}

	if _, err := os.Stat(svelteWorkdir); os.IsExist(err) {
		err = cleanDir(svelteWorkdir)
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("GoSvelt is started on [:%s]\n", addr)

	if err := gs.server.ListenAndServeTLS(addr, cert, key); err != nil {
		panic(err)
	}
}

func (gs *GoSvelt) Middleware(path string, fn MiddlewareFunc) {
	gs.middlewares[path] = fn
}

func (gs *GoSvelt) SvelteMiddleware(path string, fn SvelteMiddlewareFunc) {
	gs.svelteMiddlewares[path] = fn
}

func (gs *GoSvelt) Get(path string, h HandlerFunc) {
	gs.add(MGet, path, h)
}

func (gs *GoSvelt) Post(path string, h HandlerFunc) {
	gs.add(MPost, path, h)
}

func (gs *GoSvelt) Put(path string, h HandlerFunc) {
	gs.add(MPut, path, h)
}

func (gs *GoSvelt) Delete(path string, h HandlerFunc) {
	gs.add(MDelete, path, h)
}

func (gs *GoSvelt) Connect(path string, h HandlerFunc) {
	gs.add(MConnect, path, h)
}

func (gs *GoSvelt) Options(path string, h HandlerFunc) {
	gs.add(MOptions, path, h)
}

func (gs *GoSvelt) Static(path, file string) {
	gs.addStatic(MGet, path, file)
}

// // see config
// type SseConfig struct {
// 	Datach  chan interface{} // needed
// 	Closech chan struct{}    // needed
// 	Timeout time.Duration    // optional
// }

// sse event
type SseEvent struct {
	Name string // event name
	Data string // event datas
}

func (gs *GoSvelt) Sse(path string, datach chan interface{}, closech chan struct{}, fn func()) {
	handler := func(c *fasthttp.RequestCtx) {
		// cors headers
		//c.Response.Header.Add("Access-Control-Allow-Origin", "*")
		//c.Response.Header.Add("Access-Control-Allow-Headers", "Content-Type")
		//c.Response.Header.Add("Access-Control-Allow-Credentials", "true")

		// sse headers
		c.Response.Header.Add("Content-Type", "text/event-stream")
		c.Response.Header.Add("Transfer-Encoding", "chunked")
		c.Response.Header.Add("Cache-Control", "no-cache")
		c.Response.Header.Add("Connection", "keep-alive")

		// write body stream
		c.Response.SetBodyStream(
			fasthttp.NewStreamReader(
				func(w *bufio.Writer) {
					flush := func() {
						if err := w.Flush(); err != nil {
							fmt.Printf("sse: flushing error: %v. closing http connection\n", err)
							return
						}
					}

					//Loop:
					for {
						select {
						case <-closech:
							close(datach)

							//c.Res().Header.SetConnectionClose()

							return

						case msg := <-datach:
							switch m := msg.(type) {
							case string:
								fmt.Fprintf(w, "data: %s\n\n", m)

							case SseEvent:
								fmt.Fprintf(w, "event: %s\n\n", m.Name)
								fmt.Fprintf(w, "data: %s\n\n", m.Data)

							default: // we don't care
								fmt.Fprintf(w, "data: %s\n\n", m)
							}

							flush()
						}
					}
				},
			), -1,
		)

		// start user func
		go fn()
	}

	gs.router.Handle(MGet, path, handler)
}

// help to server Svelte files to client
func (gs *GoSvelt) Svelte(
	path, svelteFile string,
	handlerFn SvelteHandlerFunc,
	options ...SvelteOption,
) {
	gs.addSvelte(
		path,       // url path
		svelteFile, // svelte file
		handlerFn,
		options...,
	)
}

func (gs *GoSvelt) add(method, path string, h HandlerFunc) {
	gs.router.Handle(method, path, gs.newHandler(h))
}

func (gs *GoSvelt) addSvelte(
	path,
	svelteFile string,
	handlerFn SvelteHandlerFunc,
	options ...SvelteOption,
) {
	// compile svelte file to compFile
	buildId, buildFolder, err := BuildSvelte(
		svelteFile,
		options...,
	)
	if err != nil {
		log.Fatal(err)
	}

	jsBundleUrl, err := url.JoinPath(path, buildId, "bundle.js")
	if err != nil {
		panic(err)
	}

	cssBundleUrl, err := url.JoinPath(path, buildId, "bundle.css")
	if err != nil {
		panic(err)
	}

	// this map gives the js and css path
	svelteMap := Map{
		"js":  jsBundleUrl,
		"css": cssBundleUrl,
	}

	// this will handle the main route
	gs.router.Handle(
		MGet,
		path,
		gs.newFrontHandler(handlerFn, svelteMap),
	)

	// this will handle the js bundle file
	gs.addStatic(
		MGet,
		svelteMap["js"].(string),
		filepath.Join(buildFolder, "bundle.js"),
	)
	// this will handle the css bundle file
	gs.addStatic(
		MGet,
		svelteMap["css"].(string),
		filepath.Join(buildFolder, "bundle.css"),
	)
}

func (gs *GoSvelt) addStatic(method, path, file string) {
	gs.router.Handle(method, path, func(ctx *fasthttp.RequestCtx) { ctx.SendFile(file) })
}

// this create an fasthttp handler
// with an front handler and an svelte path
func (gs *GoSvelt) newFrontHandler(handlerFn SvelteHandlerFunc, svelte Map) fasthttp.RequestHandler {
	return func(bctx *fasthttp.RequestCtx) {
		ctx := gs.pool.Get().(*Context) // get context from the pool

		// make an new context for fonction
		// using fasthttp request context
		ctx.update(bctx)

		defer ctx.reset()      // reset the context with nil values
		defer gs.pool.Put(ctx) // put the context back in the pool after the reset

		// middlewares
		if len(gs.svelteMiddlewares) != 0 {
			for path, mid := range gs.svelteMiddlewares {
				if strings.HasPrefix(ctx.Path(), path) || path == "*" {
					mid(handlerFn)
				}
			}
		}

		// if there are no errors handle the req
		// else use the default error handler
		if err := handlerFn(ctx, svelte); err != nil {
			gs.errHandler(bctx, err)
		}
	}
}

// most important function,
// goal is to convert HandlerFunc to fasthttp.RequestHandler
// and to handle middlewares
func (gs *GoSvelt) newHandler(h HandlerFunc) fasthttp.RequestHandler {
	return func(bctx *fasthttp.RequestCtx) {
		// make an new context for fonction
		// using fasthttp request context
		ctx := gs.pool.Get().(*Context)
		ctx.update(bctx)

		// middlewares
		if len(gs.middlewares) != 0 {
			for p, mid := range gs.middlewares {
				if strings.HasPrefix(ctx.Path(), p) || p == "*" {
					mid(h)
				}
			}
		}

		// if there are no errors handle the req
		// else use the default error handler
		if err := h(ctx); err != nil {
			gs.errHandler(bctx, err)
		}

		// reset the context with nil values
		// and put it in the pool
		ctx.reset()
		gs.pool.Put(ctx)
	}
}
