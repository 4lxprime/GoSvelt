package main

import (
	"encoding/json"
	"fmt"
	"time"

	gs "github.com/4lxprime/GoSvelt"
	"github.com/valyala/fasthttp"
)

func ErrorHandler(c *fasthttp.RequestCtx, err error) {
	statusCode := 500

	respBytes, err := json.Marshal(struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
	}{
		Status: statusCode,
		Error:  err.Error(),
	})
	if err != nil {
		return
	}

	c.SetBody(respBytes)
	c.SetStatusCode(statusCode)
}

func main() {
	app := gs.New(
		gs.WithLog,
		gs.WithErrorHandler(ErrorHandler),
	)

	app.SvelteMiddleware("/", func(next gs.SvelteHandlerFunc) gs.SvelteHandlerFunc {
		return func(c *gs.Context, svelte gs.Map) error {
			return next(c, svelte)
		}
	})

	app.Static("/svelte_logo", "assets/svelte_logo.svg")

	app.Get("/sse", func(c *gs.Context) error {
		datach := make(chan interface{})
		closech := make(chan struct{})

		return c.Sse(datach, closech, func() {
			defer close(closech)
			datach <- "hello world"

			for i := 0; i < 6; i++ {
				time.Sleep(200 * time.Millisecond)
				datach <- gs.SseEvent{
					Name: "date",
					Data: fmt.Sprintf("time: %v", time.Now()),
				}
			}
		})
	})

	datach := make(chan interface{})
	closech := make(chan struct{})

	app.Sse("/ssetoo", datach, closech, func() {
		defer close(closech)
		datach <- "hello world"

		for i := 0; i < 6; i++ {
			time.Sleep(200 * time.Millisecond)
			datach <- gs.SseEvent{
				Name: "date",
				Data: fmt.Sprintf("time: %v", time.Now()),
			}
		}
	})

	app.Svelte("/ssepage", "views/sse/App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "assets/index.html", svelte)
		},
		gs.WithPackageManager("pnpm"),
	)

	app.Svelte("/", "App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "assets/index.html", svelte)
		},
		gs.WithPackageManager("pnpm"),
		gs.WithTailwindcss,
		gs.WithRoot("views"),
	)

	app.Start(":8080")
}
