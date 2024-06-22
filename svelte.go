package gosvelt

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/4lxprime/gitdl"
)

const (
	svelteEnv     = "./.svelte_env"
	svelteWorkdir = "./.svelte_workdir"
	svelteApp     = "App.svelte"
)

func pathFromSvelteEnv(path string) string {
	return filepath.Join(svelteEnv, path)
}

func execFromSvelteEnv(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	cmd.Dir = svelteEnv

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

var (
	errNpxNotFound = fmt.Errorf("svelte: npx is not available on your system, please install it")
	errPMNotFound  = func(packageManager string) error {
		return fmt.Errorf("svelte: %s is not available on your system, please install it", packageManager)
	}
	errPMDegit = func(packageManager string) error {
		return fmt.Errorf("svelte: %s cannot install degit on your system, if you are on linux, you can try to install it manually with 'sudo %s install -g degit' in the directory %s", packageManager, packageManager, svelteEnv)
	}
	errPMRollup = func(packageManager string) error {
		return fmt.Errorf("svelte: %s cannot install rollup on your system, if you are on linux, you can try to install it manually with 'sudo %s install -g rollup' in the directory %s", packageManager, packageManager, svelteEnv)
	}
	errPMI = func(packageManager string) error {
		return fmt.Errorf("svelte: %s cannot install needed dependencies on your system, if you are on linux, may you can try to install it manually with '%s i' in the directory %s", packageManager, packageManager, svelteEnv)
	}
	errCustomMainTs     = fmt.Errorf("svelte: cannot write custom app in %s", pathFromSvelteEnv("/src/main.ts"))
	errCustomViteEnvTs  = fmt.Errorf("svelte: cannot write custom global.d.ts")
	errCustomGlobaldTs  = fmt.Errorf("svelte: cannot write custom global.d.ts")
	errNoDefaultApp     = fmt.Errorf("svelte: no default app found (%s)", svelteApp)
	errNpxRollupCompile = fmt.Errorf("svelte: cannot compile %s with rollup, maybe you have a error in your svelte files, you may also have tried to use gs.Svelte('/path', '/your/app.svelte', ...) but it seems that app.svelte requires a parent file and to fix this, you can try using gs.AdvancedSvelte() instead", svelteEnv)
	errViteCompile      = func(packageManager string) error {
		return fmt.Errorf("svelte: vite: cannot compile %s, you may have an error in your svelte files or maybe you did give the wrong file path. you can get the full error by doing `%s run build` in %s", svelteEnv, packageManager, svelteEnv)
	}
	errCustomTailwind = fmt.Errorf("svelte: cannot write custom postcss config in %s", pathFromSvelteEnv("/postcss.config.js"))
	errCustomPostcss  = fmt.Errorf("svelte: cannot write custom tailwindcss config in %s", pathFromSvelteEnv("/tailwind.config.js"))
	errTailwinsBuild  = fmt.Errorf("svelte: there are an error during the tailwindcss compilation with postcss")
)

type SvelteOptions struct {
	tailwindcss    bool
	packageManager string
	rootFolder     *string
}
type SvelteOption func(*SvelteOptions)

var (
	WithTailwindcss = func(o *SvelteOptions) {
		o.tailwindcss = true
	}
	WithPackageManager = func(packageManager string) SvelteOption {
		return func(o *SvelteOptions) {
			o.packageManager = packageManager
		}
	}
	WithRoot = func(rootFolder string) SvelteOption {
		return func(o *SvelteOptions) {
			o.rootFolder = &rootFolder
		}
	}
)

func BuildSvelte(inputSvelteFile string, options ...SvelteOption) (string, string, error) {
	opts := new(SvelteOptions)

	for _, opt := range options {
		opt(opts)
	}

	// init svelte env ->

	if err := initSvelteEnv(opts); err != nil {
		return "", "", err
	}

	// copy user svelte files to build env ->

	if err := copySvelteFiles(inputSvelteFile, opts); err != nil {
		return "", "", err
	}

	// get build unique id and break if already exists ->

	buildHash, err := calculateTreeHash(filepath.Join(svelteEnv, "src", "app"))
	if err != nil {
		return "", "", err
	}

	buildId := fmt.Sprintf("b%s", buildHash[:8])
	buildFolder := filepath.Join(svelteWorkdir, buildId, "bundle")

	if _, err := os.Stat(filepath.Join(buildFolder, "bundle.js")); !os.IsNotExist(err) {
		return buildId, buildFolder, nil // return if already compiled
	}

	if err := os.MkdirAll(buildFolder, 0755); err != nil {
		return "", "", err
	}

	// parse and install every project modules in build env + basic setup ->
	// (the slow function)

	if err := installSvelteModules(opts); err != nil {
		return "", "", err
	}

	// build svelte environment ->

	if err := buildSvelteEnv(buildFolder, opts); err != nil {
		return "", "", err
	}

	// clear svelte environment ->

	if err := clearSvelteEnv(opts); err != nil {
		return "", "", err
	}

	return buildId, buildFolder, nil
}

func initSvelteEnv(opts *SvelteOptions) error {
	// 1st step: check that everything is ready to use on the system
	if _, err := exec.LookPath(opts.packageManager); err != nil {
		return errPMNotFound(opts.packageManager)
	}

	// 2nd step: download and init the template

	// check is svelteEnv exist, else create it
	if _, err := os.Stat(svelteEnv); os.IsNotExist(err) {
		if err := os.MkdirAll(svelteEnv, 0755); err != nil {
			return err
		}
	}

	// check is svelteWorkdir exist, else create it
	if _, err := os.Stat(svelteWorkdir); os.IsNotExist(err) {
		if err := os.MkdirAll(svelteWorkdir, 0755); err != nil {
			return err
		}
	}

	if fs, err := os.ReadDir(svelteEnv); err != nil {
		return err

	} else if len(fs) < 1 { // empty svelte_env
		// downloading the svelte vite typescript template dirrectly
		// from the official vitejs/vite repo with custom git folder download
		if err := gitdl.DownloadGit(
			"vitejs/vite",
			"packages/create-vite/template-svelte-ts",
			svelteEnv,
			gitdl.WithExclusions(
				"src/*",
				"public/*",
				".vscode",
				"*.md",
			),
			gitdl.WithLogs,
		); err != nil {
			return err
		}

	} else { // svelte_env already initialized, clean
		if err := clearSvelteEnv(opts); err != nil {
			return err
		}
	}

	// writing the vite-env.d.ts, which will be used
	// to reference typing
	if err := os.WriteFile(
		pathFromSvelteEnv("/src/vite-env.d.ts"),
		[]byte(`/// <reference types="svelte" />
/// <reference types="vite/client" />`),
		0644,
	); err != nil {
		return errCustomViteEnvTs
	}

	if opts.tailwindcss {
		// writing tailwindcss config to vite config
		if err := os.WriteFile(
			pathFromSvelteEnv("vite.config.ts"),
			[]byte(`import tailwind from'tailwindcss';import autoprefixer from'autoprefixer';import{defineConfig}from'vite';import{svelte}from'@sveltejs/vite-plugin-svelte';export default defineConfig({plugins:[svelte()],css:{postcss:{plugins:[tailwind({content:["./**/*.svelte"]}),autoprefixer]}}});`),
			0644,
		); err != nil {
			return fmt.Errorf("svelte: error while writing custom tailwindcss/postcss config (%s)", pathFromSvelteEnv("vite.config.ts"))
		}

	} else {
		// writing default vite config
		if err := os.WriteFile(
			pathFromSvelteEnv("vite.config.ts"),
			[]byte(`import{defineConfig}from'vite';import{svelte}from'@sveltejs/vite-plugin-svelte';export default defineConfig({plugins:[svelte()]});`),
			0644,
		); err != nil {
			return fmt.Errorf("svelte: error while writing custom tailwindcss/postcss config (%s)", pathFromSvelteEnv("vite.config.ts"))
		}
	}

	return nil
}

func copySvelteFiles(inputSvelteFile string, opts *SvelteOptions) error {
	var rootFolder string

	if opts.rootFolder == nil {
		rootFolder = "./"

	} else {
		rootFolder = *opts.rootFolder
	}

	inputSvelteAppFile := filepath.Join(rootFolder, inputSvelteFile)

	if _, err := os.Stat(inputSvelteAppFile); os.IsNotExist(err) {
		return fmt.Errorf("svelte: default app not found (%s)", inputSvelteAppFile)
	}

	svelteAppFolder := filepath.Join(svelteEnv, "src", "app")
	svelteAppFile := svelteApp

	if opts.rootFolder == nil { // if there is no root folder
		if err := copyFile(
			inputSvelteAppFile,
			filepath.Join(svelteAppFolder, svelteAppFile),
		); err != nil {
			return err
		}

	} else {
		if err := copyDir(
			rootFolder,
			svelteAppFolder,
		); err != nil {
			return err
		}

		// set new svelte app file because we'll move it
		// (based on the env relative svelte main file path and
		// the default svelte app file name)
		svelteAppFile = filepath.Join(filepath.Dir(inputSvelteFile), svelteApp)

		if filepath.Base(inputSvelteFile) != svelteApp {
			if err := os.Rename(
				filepath.Join(svelteAppFolder, inputSvelteFile), // e.g. .svelte_env/src/app/index.svelte
				filepath.Join(svelteAppFolder, svelteAppFile),   // e.g. .svelte_env/src/app/App.svelte
			); err != nil {
				return err
			}
		}
	}

	// writing the main.ts file, which will be used
	// to give svelte files to the vite compiler
	if err := os.WriteFile(
		pathFromSvelteEnv("src/main.ts"),
		[]byte(fmt.Sprintf(
			"import App from './app/%s'; export default new App({ target: document.body });",
			filepath.ToSlash(svelteAppFile),
		)),
		0644,
	); err != nil {
		return errCustomMainTs
	}

	return nil
}

func installSvelteModules(opts *SvelteOptions) error {
	if err := execFromSvelteEnv(opts.packageManager, "i"); err != nil {
		return errPMI(opts.packageManager)
	}

	if opts.tailwindcss {
		// install needed tailwindcss deps
		if err := execFromSvelteEnv(
			opts.packageManager,
			"install",
			"tailwindcss",
			"postcss",
			"autoprefixer",
		); err != nil {
			return errPMI(opts.packageManager)
		}
	}

	return moduleParser(opts)
}

func buildSvelteEnv(outputFolder string, opts *SvelteOptions) error {
	if err := execFromSvelteEnv(opts.packageManager, "run", "build"); err != nil {
		return errViteCompile(opts.packageManager)
	}

	assetFolder := filepath.Join(svelteEnv, "dist", "assets", "")

	matches, err := filepath.Glob(filepath.Join(assetFolder, "index-*.*"))
	if err != nil {
		return err
	}

	var noCss bool

	if len(matches) != 2 {
		if len(matches) != 1 {
			return fmt.Errorf("svelte: could not found generated bundles, builder error")
		}

		noCss = true
	}

	var jsFile string
	var cssFile string

	if filepath.Ext(matches[0]) == ".js" {
		jsFile = matches[0]
		if !noCss {
			cssFile = matches[1]
		}

	} else { // else noCss won't match so no need to check in !noCss
		jsFile = matches[1]
		cssFile = matches[0]
	}

	// copy js index to the bundle output
	if err := copyFile(
		jsFile,
		filepath.Join(outputFolder, "bundle.js"),
	); err != nil {
		return err
	}

	// copy css index to the bundle output
	if !noCss {
		if err := copyFile(
			cssFile,
			filepath.Join(outputFolder, "bundle.css"),
		); err != nil {
			return err
		}

	} else { // writing empty css bundle file
		file, err := os.Create(filepath.Join(outputFolder, "bundle.css"))
		if err != nil {
			return nil
		}
		file.Close()
	}

	return nil
}

func clearSvelteEnv(opts *SvelteOptions) error {
	_ = opts

	if err := cleanDir(filepath.Join(svelteEnv, "dist")); err != nil {
		return err
	}

	if err := cleanDir(filepath.Join(svelteEnv, "src")); err != nil {
		return err
	}

	if opts.tailwindcss {
		if err := os.WriteFile(
			pathFromSvelteEnv("vite.config.ts"),
			[]byte(`import{defineConfig}from'vite';import{svelte}from'@sveltejs/vite-plugin-svelte';export default defineConfig({plugins:[svelte()]});`),
			0644,
		); err != nil {
			return fmt.Errorf("svelte: error while writing custom tailwindcss/postcss config (%s)", pathFromSvelteEnv("vite.config.ts"))
		}
	}

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
		if !strings.Contains(mod, "/") {
			return nil
		}

		if strings.Split(mod, "/")[0] == "svelte" {
			// so just remove module
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
func moduleParser(opts *SvelteOptions) error {
	return filepath.Walk(filepath.Join(svelteEnv, "src"), func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ok, err := isFile(path); !ok && err != nil {
			panic(err)
		}

		fileExt := filepath.Ext(path)

		if fileExt != ".svelte" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		modules := parseSvelte(string(data))
		if len(modules) == 0 {
			return nil
		}

		modulesString := strings.Join(modules, " ")

		if err := execFromSvelteEnv(opts.packageManager, "install", modulesString); err != nil {
			return errPMI(opts.packageManager)
		}

		return nil
	})
}
