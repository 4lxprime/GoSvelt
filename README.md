# GoSvelt
 the `fasthttp` `fullstack` golang framwork using `svelt`.

## why gosvelt ?
### fullstack integration of svelte
```golang
func main() {
	gs := gosvelt.New()

    // here this handler will compile svelte code with related files
    // and will return the compiled page with html file
	gs.Svelte("/", "./cmd/static/App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
        // here you can also write plain html 
        // and give custom arguments instead of "svelte"
		return c.Html(200, "./cmd/static/index.html", svelte)
	})

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
### SSR:
 - we want to add an integration of server prerendering, this aim to improve performance and add css engines
### SSE:
 - an Server-Sent Event support
### Css engines like Tailwindcss
