package gosvelt

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/buaazp/fasthttprouter"
	"github.com/dgrr/http2"
	"github.com/valyala/fasthttp"
)

type Map map[string]interface{}

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
	gs.add(http.MethodGet, path, h)
}

func (gs *GoSvelt) Post(path string, h HandlerFunc) {
	gs.add(http.MethodPost, path, h)
}

func (gs *GoSvelt) Put(path string, h HandlerFunc) {
	gs.add(http.MethodPut, path, h)
}

func (gs *GoSvelt) Delete(path string, h HandlerFunc) {
	gs.add(http.MethodDelete, path, h)
}

func (gs *GoSvelt) Connect(path string, h HandlerFunc) {
	gs.add(http.MethodConnect, path, h)
}

func (gs *GoSvelt) Options(path string, h HandlerFunc) {
	gs.add(http.MethodOptions, path, h)
}

func (gs *GoSvelt) add(method, path string, h HandlerFunc) {
	gs.router.Handle(method, path, gs.newHandler(h))
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

		ctx.reset()
		gs.pool.Put(ctx)
	}
}
