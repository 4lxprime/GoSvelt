package main

import (
	"fmt"
	"time"

	gs "github.com/4lxprime/GoSvelt"
	"github.com/fasthttp/websocket"
)

func main() {
	r := gs.New()

	r.SvelteMiddleware("/", func(next gs.SvelteHandlerFunc) gs.SvelteHandlerFunc {
		return func(c *gs.Context, svelte gs.Map) error {
			return next(c, svelte)
		}
	})

	r.Get("/gg/:name", func(c *gs.Context) error {
		return c.Json(200, gs.Map{"gg": c.Param("name")})
	})

	r.Get("/test", func(c *gs.Context) error {
		return c.Text(200, "Hello, World!")
	})

	r.Get("/ws", func(c *gs.Context) error {
		return c.Ws(func(conn *websocket.Conn) {
			conn.WriteJSON(gs.Map{"ez": "pz"})
		})
	})

	r.Static("/svelte_logo", "./cmd/static/svelte_logo.svg")

	datach := make(chan interface{})
	closech := make(chan struct{})
	r.Sse("/test/sse", datach, closech, "test", func() {
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

			for i := 0; i < 2; i++ {
				time.Sleep(2 * time.Millisecond)
				datach <- gs.SseEvent{
					Name: "date",
					Data: fmt.Sprintf("%d -> actual time is %v", i, time.Now()),
				}
			}

			close(closech)
		})
	})

	r.AdvancedSvelte(
		"/ssepage",
		"cmd/static/",
		"sse/App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "./cmd/static/index.html", svelte)
		},
		gs.SvelteConfig{
			Typescript:  false,
			Tailwindcss: false,
			Pnpm:        true,
		},
	)

	// r.AdvancedSvelte("/", "cmd/static/", "app/App.svelte", func(c *gs.Context, svelte gs.Map) error {
	// 	return c.Html(200, "./cmd/static/index.html", svelte)

	// }, gs.SvelteConfig{
	// 	Typescript:  false,
	// 	Tailwindcss: true,
	// 	Pnpm:        true,
	// })

	r.Start(":8080")
}
