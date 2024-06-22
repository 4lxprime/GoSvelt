# GoSvelt
 The `fasthttp` `fullstack` golang framwork using `svelte` (support tailwindcss)
 that use to be (blazingly) faster than default `sveltekit`

## Why GoSvelt ?
### Fullstack integration of Svelte
 Yeah, gosvelt will compile, group, and serve svelte pages at runtime which is pretty cool.  
 We are using the vitejs/vite svelte typescript compiler, with this, we can do likely everything we want, we could add few really interesting options.  
 The "compiler" accept for the moment javascript / typescript svelte and tailwindcss, if you want some features to be added, i'll be happy to add them.  
 A Svelte handler will give you a **svelte map** wich contain "js" and "css" URLs, you can add to this map your own attributes that will be rendered on the html template (Note: if you add for example a "test" element to the map, you have to add the `&{test}` element in the html template)
```golang
func main() {
	app := gosvelt.New()

	app.Svelte("/", "App.svelte",
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "assets/index.html", svelte)
		},
		gs.WithPackageManager("pnpm"),
		gs.WithTailwindcss,
		gs.WithRoot("views"),
	)

	app.Start(":80")
}
```
You can note that this could be faster (blazingly fast) than default sveltekit or default vite server as go is likely way faster than nodejs.  
### Cool way to do SSE
 There are actually two way to use sse in gosvelt:  
 - The **context** way where you can instantiate your channels in the handler function and you can return a goroutine that will handle the sse stream.
 - The **handler** way wich is a handler that will take outside channels (it could be really nice if you have some external struct for events handling) and instead of giving a handler function, you just give the goroutine function that will handle the sse stream.
```golang
func main() {
	app := gosvelt.New()

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

	app.Start(":80")
}
```
### Pretty simple syntax
 The syntax is really easy to remember / use if you are beggining with golang framworks and if you already know all this (useless) framworking stuff, it's like most popular framworks (fiber, gin, echo, ...) so you won't be lost!
```golang
func main() {
	app := gosvelt.New(
		gs.WithHttp2,
	)

	app.Get("/gg/:name", func(c *gosvelt.Context) error { // url params
		return c.Json(200, gosvelt.Map{"gg": c.Param("name")})
	})

	app.Get("/ws", func(c *gosvelt.Context) error { // websocket handler
		return c.Ws(func(conn *websocket.Conn) {
			conn.WriteJSON(gosvelt.Map{"ez": "pz"})
		})
	})

	app.Static("/index", "assets/index.html") // static files

	app.Svelte("/", "views/App.svelte", // svelte page handler (runtime compiled)
		func(c *gs.Context, svelte gs.Map) error {
			return c.Html(200, "assets/index.html", svelte)
		},
	)

	app.Start(":80")
}
```
## Todo:
 - [ ] error handler panic issue
 - [ ] new gosvelt config options
 - [ ] live reload
 - [ ] template and init util (with gitdl)
 - [x] **CSR** (Client Side Rendering)
 - [ ] **SSR** (Server Side Rendering)
 - [ ] **ISR** (Incremental Static Regeneration)
 - [x] **SSE** (Server Sent Events)
 - [x] **WS** (Web Socket)
 - [x] **CSS Engine** (Tailwindcss)
 - [ ] Add layout system
