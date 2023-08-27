# GoSvelt
 the `fasthttp` `fullstack` golang framwork using `svelte` (support tailwindcss).
 just more 10 time faster than `sveltekit`

## why gosvelt ?
### fullstack integration of svelte
```golang
func main() {
	r := gosvelt.New()

	r.Svelte("/", "./static/App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
		return c.Html(200, "./static/index.html", svelte)
	})

	r.AdvancedSvelte("/adv", "./static/", "App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
		return c.Html(200, "./static/index.html", svelte)

	}, gs.SvelteConfig{
		Typescript:  false,
		Tailwindcss: true,
		Pnpm:        true,
	})

	r.Start(":80")
}
```
### cool way to made sse
```golang
	r := gosvelt.New()

	r.Get("/sse", func(c *gs.Context) error {
		datach := make(chan interface{})
		closech := make(chan struct{})

		return c.Sse(datach, closech, "test", func() {
			datach <- "hello"

			for i := 0; i < 10; i++ {
				time.Sleep(100 * time.Millisecond)
				datach <- fmt.Sprintf("%d -> actual time is %v", i, time.Now())
			}

			close(closech)
		})
	})

	datach := make(chan interface{})
	closech := make(chan struct{})

	r.Sse("/sse2", datach, closech, "test2", func() {
		datach <- "hello"

		for i := 0; i < 4; i++ {
			time.Sleep(time.Second)
			datach <- fmt.Sprintf("%d -> actual time is %v", i, time.Now())
		}

		close(closech)
	})

	r.Start(":80")
```
### pretty simple syntax
```golang
func main() {
	r := gosvelt.New()

	r.Get("/gg/:name", func(c *gosvelt.Context) error {
		return c.Json(200, gosvelt.Map{"gg": c.Param("name")})
	})

	r.Get("/ws", func(c *gosvelt.Context) error {
		return c.Ws(func(conn *websocket.Conn) {
			conn.WriteJSON(gosvelt.Map{"ez": "pz"})
		})
	})

	r.Static("/index", "./cmd/static/index.html")

	r.Svelte("/", "./cmd/static/App.svelte", func(c *gosvelt.Context, svelte gosvelt.Map) error {
		return c.Html(200, "./cmd/static/index.html", svelte)
	})

	r.Start(":80")
}
```
## todo:
 - [x] **CSR** (Client Side Rendering)
 - [ ] **SSR** (Server Side Rendering)
 - [ ] **ISR** (Incremental Static Regeneration)
 - [x] **SSE** (Server Sent Events)
 - [x] **WS** (Web Socket)
 - [x] **CSS Engine** (Tailwindcss)
