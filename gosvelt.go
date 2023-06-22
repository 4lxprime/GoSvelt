package gosvelt

import (
	"fmt"
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

// todo: add handle methods
// + add newHandler function that transform HandlerFunc into fasthttp.RequestHandler
