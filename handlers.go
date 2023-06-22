package gosvelt

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

var defaultErrorHandler = func(c *fasthttp.RequestCtx, err error) {
	fmt.Printf("Error occurred: %v\n", err)

	c.Write([]byte(err.Error()))
}

type HandlerFunc func(c *Context) error
type SvelteHandlerFunc func(c *Context, svelte Map) error
type ErrorHandlerFunc func(c *fasthttp.RequestCtx, err error)
