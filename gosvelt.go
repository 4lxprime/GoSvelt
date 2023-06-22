package gosvelt

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/buaazp/fasthttprouter"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
)

type Map map[string]interface{}

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

type Config struct {
	Http2 bool
}

type GoSvelt struct {
	Config     *Config
	server     *fasthttp.Server
	router     *fasthttprouter.Router
	pool       sync.Pool
	errHandler ErrorHandlerFunc
}

func New(cfg ...*Config) *GoSvelt {
	gs := &GoSvelt{
		Config:     &Config{},
		server:     &fasthttp.Server{},
		router:     fasthttprouter.New(),
		pool:       sync.Pool{},
		errHandler: defaultErrorHandler,
	}

	if len(cfg) != 0 {
		gs.Config = cfg[0]
	}

	gs.router.NotFound = func(ctx *fasthttp.RequestCtx) {
		gs.errHandler(ctx, fmt.Errorf("not found"))
	}
	gs.pool.New = gs.newContext

	return gs
}

func (gs *GoSvelt) Start(addr string) {
	gs.server.Handler = gs.router.Handler

	if gs.Config.Http2 {
		http2.ConfigureServer(gs.server, http2.ServerConfig{})
	}

	err := gs.server.ListenAndServe(addr)
	if err != nil {
		panic(err)
	}
}

func (gs *GoSvelt) StartTLS(addr, cert, key string) {
	gs.server.Handler = gs.router.Handler

	if gs.Config.Http2 {
		http2.ConfigureServer(gs.server, http2.ServerConfig{})
	}

	err := gs.server.ListenAndServeTLS(addr, cert, key)
	if err != nil {
		panic(err)
	}
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
	gs.addStatic(MGet, path, newFileHandler(file))
}

// help to server Svelte files to client
func (gs *GoSvelt) Svelte(path string, svelteFile string, fh SvelteHandlerFunc, layouts ...string) {
	gs.addSvelte(path, svelteFile, fh, layouts...)
}

func (gs *GoSvelt) add(method, path string, h HandlerFunc) {
	gs.router.Handle(method, path, gs.newHandler(h))
}

func (gs *GoSvelt) addSvelte(path, file string, fh SvelteHandlerFunc, layouts ...string) {

	compName := fileName(file)

	if _, err := os.Stat(filepath.Join(svelteWorkdir, "/", compName)); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Join(svelteWorkdir, "/", compName), 0755)
		if err != nil {
			panic(err)
		}
	}

	err := compileSvelteFile(file, filepath.Join(svelteWorkdir, "/", compName, "/bundle"), layouts...)
	if err != nil {
		panic(err)
	}

	// this is for the // in start of path
	var gpath string
	if string(path[len(path)-1]) == "/" {
		gpath = path[:len(path)-1]
	}

	// this map gives the js and css path
	svelteMap := Map{
		"js":  gpath + "/bundle/bundle.js",
		"css": gpath + "/bundle/bundle.css",
	}

	// this will handle the main route
	gs.router.Handle(MGet, path, gs.newFrontHandler(fh, svelteMap))

	// this will handle the js bundle file
	gs.addStatic(MGet, gpath+"/bundle/bundle.js", newFileHandler(svelteWorkdir+"/"+compName+"/bundle.js"))
	// this will handle the css bundle file
	gs.addStatic(MGet, gpath+"/bundle/bundle.css", newFileHandler(svelteWorkdir+"/"+compName+"/bundle.css"))
}

func (gs *GoSvelt) addStatic(method, path string, h fasthttp.RequestHandler) {
	gs.router.Handle(method, path, h)
}

// NOTE: this can be used in static handlers
func newFileHandler(path string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		fasthttp.ServeFile(ctx, path)
	}
}

// this create an fasthttp handler
// with an front handler and an svelte path
func (gs *GoSvelt) newFrontHandler(h SvelteHandlerFunc, svelte Map) fasthttp.RequestHandler {
	return func(bctx *fasthttp.RequestCtx) {
		// make an new context for fonction
		// using fasthttp request context
		ctx := gs.pool.Get().(*Context)
		ctx.update(bctx)

		// if there are no errors handle the req
		// else use the default error handler
		if err := h(ctx, svelte); err != nil {
			gs.errHandler(bctx, err)
		}

		// reset the context with nil values
		// and put it in the pool
		ctx.reset()
		gs.pool.Put(ctx)
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
