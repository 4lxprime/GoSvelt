package main

import (
	gosvelt "github.com/4lxprime/GoSvelt"
	"github.com/fasthttp/websocket"
)

func main() {
	gs := gosvelt.New()

	gs.Get("/gg/:name", func(c *gosvelt.Context) error {
		return c.Json(200, gosvelt.Map{"gg": c.Param("name")})
	})

	gs.Get("/ws", func(c *gosvelt.Context) error {
		return c.Ws(func(conn *websocket.Conn) {
			conn.WriteJSON(gosvelt.Map{"ez": "pz"})
		})
	})

	gs.Static("/index", "./cmd/static/index.html")

	gs.Svelte("/", "./cmd/static/App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
		// we can also do:
		// return c.Html(200, "<!DOCTYPE html><html lang="en">...", svelte)
		// but i prefer to do it with a file
		return c.Html(200, "./cmd/static/index.html", svelte)
	})

	gs.Start(":8080")
}
