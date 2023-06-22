package gosvelt

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	svelteEnv     = "./.svelte_env"
	svelteWorkdir = "./.svelte_workdir"
	svelteApp     = "App.svelte"
)

var (
	errNpxNotFound            = fmt.Errorf("svelte: npx is not available on your system, please install it")
	errNpmDegit               = fmt.Errorf("svelte: npm cannot install degit on your system")
	errNpmRollup              = fmt.Errorf("svelte: npm cannot install rollup on your system")
	errNpmDegitSvelteTemplate = fmt.Errorf("svelte: npm cannot create degit svelte template on your system")
	errNpmI                   = fmt.Errorf("svelte: npm cannot install needed dependencies on your system")
	errCustomMainjs           = fmt.Errorf("svelte: cannot write custom app in %s", svelteEnv+"/src/main.js")
	errNoDefaultApp           = fmt.Errorf("svelte: no default app found (%s)", svelteApp)
	errLayoutsCannotBeApp     = fmt.Errorf("svelte: layouts cannot be %s", svelteApp)
	errNpxRollupCompile       = fmt.Errorf("svelte: cannot compile %s", svelteEnv)
)

// this build the .svelte_env file with npm
// NOTE: this requiere node installed
// NOTE: theses functions can be slower because
// they do not affect requests handling
func newSvelteEnv() error {
	tempMsg := "Svelte environment being created..."
	tempChan := make(chan struct{})

	go temporaryText(tempChan, tempMsg)

	if _, err := os.Stat(svelteEnv); os.IsNotExist(err) {
		err := os.MkdirAll(svelteEnv, 0755)
		if err != nil {
			return err
		}
		// check is svelteWorkdir exist, else create it
		if _, err := os.Stat(svelteWorkdir); os.IsNotExist(err) {
			err := os.MkdirAll(svelteWorkdir, 0755)
			if err != nil {
				return err
			}
		}
		_, err = exec.LookPath("npx")
		if err != nil {
			return errNpxNotFound
		}

		// idk how to check if an npm dep exist so
		// i install dep before all launches
		if err := exec.Command("npm", "install", "-g", "degit").Run(); err != nil {
			return errNpmDegit
		}
		if err := exec.Command("npm", "install", "-g", "rollup").Run(); err != nil {
			return errNpmRollup
		}

		cmd := exec.Command("npx", "degit", "sveltejs/template", ".svelte_env")
		cmd.Dir = "./"
		if err := cmd.Run(); err != nil {
			return errNpmDegitSvelteTemplate
		}

		cmd = exec.Command("npm", "i")
		cmd.Dir = svelteEnv
		if err := cmd.Run(); err != nil {
			return errNpmI
		}

		// custom app main.js
		err = ioutil.WriteFile(svelteEnv+"/src/main.js", []byte("import App from './"+svelteApp+"'; export default new App({ target: document.body });"), 0644)
		if err != nil {
			return errCustomMainjs
		}
	}

	close(tempChan)
	time.Sleep(100 * time.Millisecond)

	return nil
}

// this transform svelte file to js and css bundle
// NOTE: in outFile, don't give an file ext like .js
// NOTE: theses functions can be slower because
// they do not affect requests handling
func compileSvelteFile(inFile, outFile string, layouts ...string) error {
	// check is svelte_env exist
	if _, err := os.Stat(svelteEnv); os.IsNotExist(err) {
		if err := newSvelteEnv(); err != nil {
			return err
		}
	}

	tempMsg := "Svelte compilation in progress..."
	tempChan := make(chan struct{})

	go temporaryText(tempChan, tempMsg)

	isFile, err := isFile(inFile)
	if err != nil {
		return err
	}

	// check if inFile is an svelte file or an directory
	if isFile {
		// move svelte root path inFile to env
		if err := copyDir(filepath.Dir(inFile), filepath.Join(svelteEnv, "/src/")); err != nil {
			return err
		}

		// then if there are not App.svelte, rename inFile into App.svelte
		if file(inFile) != svelteApp {
			if err := copyFile(inFile, filepath.Join(svelteEnv, "/src/", svelteApp)); err != nil {
				return err
			}
		}
	} else {
		// move svelte root path inFile to env
		if err := copyDir(inFile, filepath.Join(svelteEnv, "/src/")); err != nil {
			return err
		}

		if !exist(filepath.Join(svelteEnv, "/src/", svelteApp)) {
			return errNoDefaultApp
		}
	}

	// move layouts to ./.svelte_env/src/
	if len(layouts) != 0 {
		for _, layout := range layouts {
			lsName := file(layout)

			if lsName == svelteApp {
				return errLayoutsCannotBeApp
			}

			err = copyFile(layout, filepath.Join(svelteEnv, "/src/", lsName))
			if err != nil {
				return err
			}
		}
	}

	// compile env with rollup command
	cmd := exec.Command("npx", "rollup", "-c")
	cmd.Dir = svelteEnv
	if err := cmd.Run(); err != nil {
		return errNpxRollupCompile
	}

	// move js bundle file to outFile
	if err := copyFile(svelteEnv+"/public/build/bundle.js", outFile+".js"); err != nil {
		return err
	}

	// move css bundle file to outFile
	if err := copyFile(svelteEnv+"/public/build/bundle.css", outFile+".css"); err != nil {
		return err
	}

	err = cleanDir(filepath.Join(svelteEnv, "/src/"))
	if err != nil {
		return err
	}

	// recreate src/main.js because of clean
	err = ioutil.WriteFile(svelteEnv+"/src/main.js", []byte("import App from './"+svelteApp+"'; export default new App({ target: document.body });"), 0644)
	if err != nil {
		return errCustomMainjs
	}

	close(tempChan)
	time.Sleep(100 * time.Millisecond)

	return nil
}
