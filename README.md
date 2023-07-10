# GoSvelt
 the `fasthttp` `fullstack` golang framwork using `svelte` (support tailwindcss).

## why gosvelt ?
### fullstack integration of svelte
```golang
func main() {
	gs := gosvelt.New()

    // here this handler will compile svelte code with related files
    // and will return the compiled page with html file
	gs.Svelte("/", "./static/App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
        // here you can also write plain html 
        // and give custom arguments instead of "svelte"
		return c.Html(200, "./static/index.html", svelte)
	}, true) // true is for tailwindcss

	gs.AdvancedSvelte("/adv", "./static/", "App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
		return c.Html(200, "./static/index.html", svelte)
	}, true)

	gs.Start(":80")
}
```
### pretty simple syntax
```golang
func main() {
	gs := gosvelt.New()

	gs.Get("/gg/:name", func(c *gosvelt.Context) error {
		return c.Json(200, gosvelt.Map{"gg": c.Param("name")})
	})

    // pretty simple way to handle ws
	gs.Get("/ws", func(c *gosvelt.Context) error {
		return c.Ws(func(conn *websocket.Conn) {
			conn.WriteJSON(gosvelt.Map{"ez": "pz"})
		})
	})

	gs.Static("/index", "./cmd/static/index.html")

	gs.Svelte("/", "./cmd/static/App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
		return c.Html(200, "./cmd/static/index.html", svelte)
	})

	gs.Start(":80")
}
```
## todo:
 - [ ] SSR
 - [ ] SSE
 - [x] CSS Engines like Tailwindcss
