package main

import (
	"encoding/json"
	"fmt"
	"time"

	gs "github.com/4lxprime/GoSvelt"
	"github.com/fasthttp/websocket"
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

	r.Get("/gg/:name", func(c *gs.Context) error {
		return c.Json(200, gs.Map{"gg": c.Param("name")})
	})

	r.Get("/test", gs.String("Hello, World!"))

	r.Get("/ws", func(c *gs.Context) error {
		return c.Ws(func(conn *websocket.Conn) {
			conn.WriteJSON(gs.Map{"ez": "pz"})
		})
	})

	r.Static("/svelte_logo", "static/svelte_logo.svg")

	datach := make(chan interface{})
	closech := make(chan struct{})
	r.Sse("/test/sse", datach, closech, func() {
		datach <- "hello"

		for i := 0; i < 4; i++ {
			time.Sleep(time.Second)
			datach <- fmt.Sprintf("%d -> actual time is %v", i, time.Now())
		}

		close(closech)
	})

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

	r.AdvancedSvelte("/ssepage", "static/", "sse/App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "static/index.html", svelte)
		},
		gs.SvelteConfig{
			Pnpm: true,
		},
	)

	r.AdvancedSvelte("/", "static/", "app/App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "static/index.html", svelte)
		},
		gs.SvelteConfig{
			Tailwindcss: true,
			Pnpm:        true,
		})

	r.Start(":8080")
}
