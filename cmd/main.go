package main

import (
	gs "github.com/4lxprime/GoSvelt"
	"github.com/fasthttp/websocket"
)

func main() {
	r := gs.New(&gs.Config{
		TypeScript: false,
	})

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

	r.AdvancedSvelte("/", "./cmd/static/", "app/App.svelte", func(c *gs.Context, svelte gs.Map) error {
		return c.Html(200, "./cmd/static/index.html", svelte)
	}, true)

	r.Start(":8080")
}
