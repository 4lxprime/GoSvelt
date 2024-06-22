package main

import (
	"encoding/json"
	"fmt"
	"time"

	gs "github.com/4lxprime/GoSvelt"
	"github.com/valyala/fasthttp"
)

func main() {
	r := gs.New(&gs.Config{
		Log:   true,
		Http2: false,
		ErrorHandler: func(c *fasthttp.RequestCtx, err error) {
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
		},
	})

	r.SvelteMiddleware("/", func(next gs.SvelteHandlerFunc) gs.SvelteHandlerFunc {
		return func(c *gs.Context, svelte gs.Map) error {
			return next(c, svelte)
		}
	})

	r.Static("/svelte_logo", "assets/svelte_logo.svg")

	r.Get("/sse", func(c *gs.Context) error {
		datach := make(chan interface{})
		closech := make(chan struct{})

		return c.Sse(datach, closech, func() {
			datach <- "hello world"

			for i := 0; i < 6; i++ {
				time.Sleep(200 * time.Millisecond)
				datach <- gs.SseEvent{
					Name: "date",
					Data: fmt.Sprintf("time: %v", time.Now()),
				}
			}

			close(closech)
		})
	})

	r.Svelte("/ssepage", "views/sse/App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "assets/index.html", svelte)
		},
		gs.WithPackageManager("pnpm"),
	)

	r.Svelte("/", "App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "assets/index.html", svelte)
		},
		gs.WithPackageManager("pnpm"),
		gs.WithTailwindcss,
		gs.WithRoot("views"),
	)

	r.Start(":8080")
}
