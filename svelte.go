package gosvelt

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

const (
	svelteEnv     = "./.svelte_env"
	svelteWorkdir = "./.svelte_workdir"
	svelteApp     = "App.svelte"
)

var (
	errNpxNotFound      = fmt.Errorf("svelte: npx is not available on your system, please install it")
	errNpmDegit         = fmt.Errorf("svelte: npm cannot install degit on your system, if you are on linux, may you can try to install it manually with 'sudo npm install -g degit' in the directory %s", svelteEnv)
	errNpmRollup        = fmt.Errorf("svelte: npm cannot install rollup on your system, if you are on linux, may you can try to install it manually with 'sudo npm install -g rollup' in the directory %s", svelteEnv)
	errNpmI             = fmt.Errorf("svelte: npm cannot install needed dependencies on your system, if you are on linux, may you can try to install it manually with 'npm i' in the directory %s", svelteEnv)
	errCustomMaints     = fmt.Errorf("svelte: cannot write custom app in %s", svelteEnv+"/src/main.ts")
	errCustomGlobaldts  = fmt.Errorf("svelte: cannot write custom global.d.ts")
	errNoDefaultApp     = fmt.Errorf("svelte: no default app found (%s)", svelteApp)
	errNpxRollupCompile = fmt.Errorf("svelte: cannot compile %s with rollup, maybe you have an error in your svelte files, you may also have tried to use gs.Svelte('/path', '/your/app.svelte', ...) but it seems that app.svelte requires a parent file and to fix this, you can try using gs.AdvancedSvelte() instead", svelteEnv)
	errCustomTailwind   = fmt.Errorf("svelte: cannot write custom tailwindcss config in %s", svelteEnv+"/postcss.config.js")
	errCustomPostcss    = fmt.Errorf("svelte: cannot write custom postcss config in %s", svelteEnv+"/tailwind.config.js")
	errTailwinsBuild    = fmt.Errorf("svelte: there are an error during the tailwindcss compilation with postcss")
)

// svelte compilator config
type SvelteConfig struct {
	Typescript  bool // default: false
	Tailwindcss bool // default: false
	Pnpm        bool // default: false
}

// this build the .svelte_env file with npm
//
//	like this
//	err := gs.newSvelteEnv(SvelteConfig{...})
//
// NOTE: this requiere nodejs installed
// NOTE: theses functions can be slower because
// they do not affect requests handling
func (gs *GoSvelt) newSvelteEnv(cfg SvelteConfig) error {
	tempMsg := "Svelte environment being created..."
	tempChan := make(chan struct{})

	go temporaryText(tempChan, tempMsg)

	if _, err := os.Stat(svelteEnv); os.IsNotExist(err) {
		err := os.MkdirAll(svelteEnv, 0755)
		if err != nil {
			return err
		}
	}
	// check is svelteWorkdir exist, else create it
	if _, err := os.Stat(svelteWorkdir); os.IsNotExist(err) {
		err := os.MkdirAll(svelteWorkdir, 0755)
		if err != nil {
			return err
		}
	}
	_, err := exec.LookPath("npx")
	if err != nil {
		return errNpxNotFound
	}

	var url string
	if cfg.Typescript {
		url = "https://github.com/4lxprime/svelteTsTemplate"

	} else {
		url = "https://github.com/4lxprime/svelteJsTemplate"
	}

	_, err = git.PlainClone(svelteEnv, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return fmt.Errorf("error during sveltejs/template clone (%s)", err)
	}

	var cmd *exec.Cmd
	if cfg.Pnpm {
		cmd = exec.Command("pnpm", "i")

	} else {
		cmd = exec.Command("npm", "i")
	}
	cmd.Dir = svelteEnv
	if err := cmd.Run(); err != nil {
		return errNpmI
	}

	// typescript custom main and global.d files
	if cfg.Typescript {
		// custom app main.ts
		err = os.WriteFile(svelteEnv+"/src/main.ts", []byte("import App from './"+svelteApp+"'; export default new App({ target: document.body });"), 0644)
		if err != nil {
			return errCustomMaints
		}

		// custom app global.d.ts
		err = os.WriteFile(svelteEnv+"/src/global.d.ts", []byte(`/// <reference types="svelte" />`), 0644)
		if err != nil {
			return errCustomGlobaldts
		}

	} else {
		// custom app main.js
		err = os.WriteFile(svelteEnv+"/src/main.js", []byte("import App from './"+svelteApp+"'; export default new App({ target: document.body });"), 0644)
		if err != nil {
			return errCustomMaints
		}
	}

	close(tempChan)
	time.Sleep(100 * time.Millisecond)

	return nil
}

// this transform svelte file to js and css bundle
//
//	like this:
//	err := gs.compileSvelteFile("app/App.svelte", "<svelteWorkdir>/<compName>/bundle", "views/", SvelteConfig{...})
//
// NOTE: in outFile, don't give an file ext like .js
// NOTE: theses functions can be slower because
// they do not affect requests handling
func (gs *GoSvelt) compileSvelteFile(inFile, outFile, rootDir string, cfg ...SvelteConfig) error {
	config := SvelteConfig{}

	if len(cfg) == 0 {
		// set default config
		config = SvelteConfig{
			Typescript:  false,
			Tailwindcss: false,
			Pnpm:        false,
		}
	} else {
		config = cfg[0]
	}

	// check is svelte_env exist
	if _, err := os.Stat(svelteEnv); os.IsNotExist(err) {
		if err := gs.newSvelteEnv(config); err != nil {
			return err
		}
	}

	// check if file is empty
	if fs, err := os.ReadDir(svelteEnv); len(fs) == 0 || err != nil {
		if err := gs.newSvelteEnv(config); err != nil {
			return err
		}
	}

	// check if we are on js or ts
	if _, err := os.Stat(filepath.Join(svelteEnv, "tsconfig.json")); os.IsExist(err) != config.Typescript {
		if err := cleanDir(svelteEnv); err != nil {
			return err
		}

		if err := gs.newSvelteEnv(config); err != nil {
			return err
		}
	}

	tempMsg := "Svelte compilation in progress..."
	tempChan := make(chan struct{})

	go temporaryText(tempChan, tempMsg)

	// no worries, in classic svelte handler,
	// rootdir is equal to "",
	// so its the same
	oldFile := inFile
	inFile = strings.ReplaceAll(filepath.Join(rootDir, inFile), `\`, "/") // fix the non-separated path

	isFile, err := isFile(inFile)
	if err != nil {
		return err
	}

	// check if inFile is an svelte file or an directory
	if isFile {
		// move svelte root path inFile to env
		// rootdir copy
		if err := copyDir(filepath.Dir(rootDir), filepath.Join(svelteEnv, "/src/")); err != nil {
			return err
		}

		// then if there are not App.svelte, rename inFile into App.svelte
		if file(inFile) != svelteApp {
			if err := copyFile(inFile, filepath.Join(svelteEnv, "/src/", filepath.Dir(oldFile), svelteApp)); err != nil {
				return err
			}
		}

	} else {
		// move svelte root path inFile to env
		// rootdir copy
		if err := copyDir(rootDir, filepath.Join(svelteEnv, "/src/")); err != nil {
			return err
		}

		if !exist(filepath.Join(svelteEnv, "/src/", oldFile, svelteApp)) {
			return errNoDefaultApp
		}
	}

	// custom app main.js that include custom rootDir
	if rootDir != "" {
		// if typescript
		if exist(svelteEnv + "/src/main.ts") {
			err = os.WriteFile(svelteEnv+"/src/main.ts", []byte("import App from './"+strings.ReplaceAll(filepath.Join(filepath.Dir(oldFile), svelteApp), `\`, "/")+"'; export default new App({ target: document.body });"), 0644)
			if err != nil {
				return errCustomMaints
			}

		} else {
			err = os.WriteFile(svelteEnv+"/src/main.js", []byte("import App from './"+strings.ReplaceAll(filepath.Join(filepath.Dir(oldFile), svelteApp), `\`, "/")+"'; export default new App({ target: document.body });"), 0644)
			if err != nil {
				return errCustomMaints
			}
		}
	}

	// write configs
	if config.Tailwindcss {
		// write tailwindcss custom config from
		// gosvelt config
		if len(gs.Config.TailwindcssCfg) == 0 {
			err := os.WriteFile(svelteEnv+"/tailwind.config.js", []byte(`module.exports = {purge: ["./**/*.svelte", "./**/*.html"], theme: {extend: {}}, variants: {}, plugins: []}`), 0644)
			if err != nil {
				return errCustomTailwind
			}

		} else {
			err := os.WriteFile(svelteEnv+"/tailwind.config.js", []byte(gs.Config.TailwindcssCfg), 0644)
			if err != nil {
				return errCustomTailwind
			}
		}

		// write postcss custom config from
		// gosvelt config
		if len(gs.Config.PostcssCfg) == 0 {
			err = os.WriteFile(svelteEnv+"/postcss.config.cjs", []byte(`module.exports = {plugins: [require("tailwindcss"), require("autoprefixer")]}`), 0644)
			if err != nil {
				return errCustomPostcss
			}

		} else {
			err = os.WriteFile(svelteEnv+"/postcss.config.cjs", []byte(gs.Config.PostcssCfg), 0644)
			if err != nil {
				return errCustomPostcss
			}
		}

		// install needed deps for tailwindcss
		var cmd *exec.Cmd
		if config.Pnpm {
			cmd = exec.Command("pnpm", "install", "tailwindcss", "postcss", "postcss-cli", "autoprefixer")

		} else {
			cmd = exec.Command("npm", "install", "tailwindcss", "postcss", "postcss-cli", "autoprefixer")
		}
		cmd.Dir = svelteEnv
		if err := cmd.Run(); err != nil {
			return errNpmI
		}
	}

	// parse and install all needed modules
	moduleParser(config)

	// compile env with rollup command
	cmd := exec.Command("npx", "rollup", "-c")
	cmd.Dir = svelteEnv
	if err := cmd.Run(); err != nil {
		return errNpxRollupCompile
	}

	if config.Tailwindcss {
		// build tailwindcss to our bundle
		cmd = exec.Command("npx", "postcss", "public/build/bundle.css", "-o", "public/build/bundle.css")
		cmd.Dir = svelteEnv
		if err = cmd.Run(); err != nil {
			fmt.Println(err)
			return errTailwinsBuild
		}
	}

	// move js bundle file to outFile
	if err := copyFile(svelteEnv+"/public/build/bundle.js", outFile+".js"); err != nil {
		return err
	}

	// move css bundle file to outFile
	if err := copyFile(svelteEnv+"/public/build/bundle.css", outFile+".css"); err != nil {
		return err
	}

	// clean ./.svelte_env/src/ directory
	err = cleanDir(filepath.Join(svelteEnv, "/src/"))
	if err != nil {
		return err
	}

	// rewrite because of dir clean
	if exist(svelteEnv + "/src/main.ts") {
		// custom app main.ts
		err = os.WriteFile(svelteEnv+"/src/main.ts", []byte("import App from './"+svelteApp+"'; export default new App({ target: document.body });"), 0644)
		if err != nil {
			return errCustomMaints
		}

		// custom app global.d.ts
		err = os.WriteFile(svelteEnv+"/src/global.d.ts", []byte(`/// <reference types="svelte" />`), 0644)
		if err != nil {
			return errCustomGlobaldts
		}

	} else {
		// custom app main.js
		err = os.WriteFile(svelteEnv+"/src/main.js", []byte("import App from './"+svelteApp+"'; export default new App({ target: document.body });"), 0644)
		if err != nil {
			return errCustomMaints
		}
	}

	// yeah i know, i slow the function for pretty text clean...
	close(tempChan)
	time.Sleep(100 * time.Millisecond)

	return nil
}

// this will parse a svelte file for found modules
//
//	like this:
//	mods := parseSvelte("svelte_file_read")
//
// NOTE: this remove svelte runtime modules
func parseSvelte(data string) []string {
	var mods []string

	re := regexp.MustCompile(`import\s+(\w+)\s+from\s+['"](\w+)['"]`)
	matches := re.FindAllStringSubmatch(data, -1)
	for _, match := range matches {
		mods = append(mods, match[2])
	}

	for i := len(mods) - 1; i >= 0; i-- {
		mod := mods[i]
		if ext := filepath.Ext(mod); ext == ".svelte" {
			mods = append(mods[:i], mods[i+1:]...)
		}

		// if there are / in module
		// this can be svelte/something
		// or something else
		if strings.Contains(mod, "/") {
			mods = append(mods[:i], mods[i+1:]...)
		}
	}

	return mods
}

// this will found and parse all svelte files
// for install needed modules
// because rollup compillator cannot install
// requiere module in svelte pages
//
//	like this:
//	go moduleparser(SvelteConfig{...})
func moduleParser(cfg SvelteConfig) {
	filepath.Walk(filepath.Join(svelteEnv, "src"), func(path string, info fs.FileInfo, err error) error {
		if ok, err := isFile(path); ok {
			ext := filepath.Ext(path)
			if ext == ".svelte" {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				mods := parseSvelte(string(data))
				if len(mods) != 0 {
					modules := strings.Join(mods, " ")

					var cmd *exec.Cmd
					if cfg.Pnpm {
						cmd = exec.Command("pnpm", "i", modules)

					} else {
						cmd = exec.Command("npm", "i", modules)
					}
					cmd.Dir = svelteEnv
					err = cmd.Run()
					if err != nil {
						return err
					}
				}
			}

		} else {
			if err != nil {
				panic(err)
			}
		}

		return nil
	})
}
