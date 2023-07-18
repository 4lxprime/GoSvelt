package gosvelt

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

var defaultErrorHandler = func(c *fasthttp.RequestCtx, err error) {
	fmt.Printf("[%s] -> %v\n", c.Path(), err)

	c.Write([]byte(err.Error()))
}

type HandlerFunc func(c *Context) error
type MiddlewareFunc func(next HandlerFunc) HandlerFunc
type SvelteMiddlewareFunc func(next SvelteHandlerFunc) SvelteHandlerFunc
type SvelteHandlerFunc func(c *Context, svelte Map) error
type ErrorHandlerFunc func(c *fasthttp.RequestCtx, err error)
