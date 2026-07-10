//go:build !docker
// +build !docker

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/fsnotify/fsnotify"
)

var RUNNING_IN_DOCKER = false

func initAppFS() (fs.FS, fs.FS) {
	var (
		// tplFS    = filesystem.Sub(assetsFS, "assets/templates")
		// staticFS = filesystem.Sub(assetsFS, "assets/static")
		tplFS    = os.DirFS("./mailman/assets/templates")
		staticFS = os.DirFS("./mailman/assets/static")
	)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	err = filepath.Walk("./mailman/assets/templates", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		return watcher.Add(path)

	})
	if err != nil {
		panic(err)
	}

	var quit = make(chan struct{})
	var shutdownFuncs = []func() error{
		func() error { return watcher.Close() },
		func() error {
			close(quit)
			return nil
		},
	}

	shutdownFuncs = append(shutdownFuncs, django.ConfigGet(
		django.Global.Settings, APP_SHUTDOWN, []func() error{},
	)...)

	django.Global.Settings.Set(APP_SHUTDOWN, shutdownFuncs)

	go func() {
		for {
			select {
			case <-quit:
				return
			case e, ok := <-watcher.Events:
				if !ok {
					return
				}

				fmt.Printf(
					"%s File %s%q%s changed, clearing template cache...\n",
					time.Now().Format(time.TimeOnly),
					logger.CMD_Yellow,
					filepath.ToSlash(e.Name),
					logger.CMD_Reset,
				)

				tpl.Global.(*tpl.TemplateRenderer).
					FS().(*filesystem.CacheFS[*filesystem.MultiFS]).
					Changed()
			}
		}
	}()

	return tplFS, staticFS
}
