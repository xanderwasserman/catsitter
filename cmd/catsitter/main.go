package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/xanderwasserman/catsitter"
	"github.com/xanderwasserman/catsitter/internal/app"
	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/host"
	"github.com/xanderwasserman/catsitter/internal/pet"
	"github.com/xanderwasserman/catsitter/internal/pet/cat"
	"github.com/xanderwasserman/catsitter/internal/tray"
	"github.com/xanderwasserman/catsitter/internal/update"
	"github.com/xanderwasserman/catsitter/internal/version"
)

func main() {
	log.SetFlags(log.Ltime)
	log.SetPrefix("catsitter: ")
	flags, err := config.ParseFlags(os.Args, os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		log.Fatal(err)
	}
	if flags.Version {
		fmt.Println(version.Version)
		return
	}
	path, err := config.Path()
	if err != nil {
		log.Fatal(err)
	}
	if flags.CheckUpdate || flags.Update {
		if err := runUpdateCommand(path, flags.Update); err != nil {
			log.Fatal(err)
		}
		return
	}
	saved, err := config.Load(path)
	if err != nil {
		log.Fatal(err)
	}
	live, err := flags.Apply(saved)
	if err != nil {
		log.Fatal(err)
	}
	backends := host.Open()
	defer backends.Close()
	session := app.New(live, saved, path, backends, update.NewService(updateOptions(path, live.CheckUpdates)))
	session.Start()
	defer session.Shutdown()
	bar := tray.New(session)
	defer bar.Stop()
	if err := pet.Run(cat.New(), session, bar, pet.Desktop{
		Pump:         backends.Pump,
		Overlay:      backends.Overlay,
		MenuGrab:     backends.MenuGrab,
		Pointer:      backends.Pointer,
		Press:        backends.Press,
		Place:        backends.Place,
		NoteFocus:    backends.NoteFocus,
		RestoreFocus: backends.RestoreFocus,
		SingleThread: backends.SingleThread,
	}); err != nil {
		log.Fatal(err)
	}
}

func updateOptions(configPath string, auto bool) update.Options {
	home, _ := os.UserHomeDir()
	exe, _ := os.Executable()
	return update.Options{
		Version:    version.Version,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		Releases:   update.FeedFromEnv("catsitter/" + version.Version + " (+" + control.Project + ")"),
		Launcher:   installer(),
		Store:      update.Store{Path: filepath.Join(filepath.Dir(configPath), "update.json")},
		CanInstall: update.Installed(exe, userHome(home)),
		Auto:       auto,
	}
}

func runUpdateCommand(configPath string, install bool) error {
	home, _ := os.UserHomeDir()
	t := update.Terminal{
		Options:     updateOptions(configPath, false),
		Installer:   installer(),
		InstallPath: update.InstallPath(runtime.GOOS, userHome(home)),
		Out:         os.Stdout,
	}
	if install {
		return t.Install(context.Background())
	}
	_, _, err := t.Check(context.Background())
	return err
}

func userHome(dir string) update.Home {
	return update.Home{Dir: dir, LocalAppData: os.Getenv("LOCALAPPDATA")}
}

func installer() update.Installer {
	if runtime.GOOS == "windows" {
		return update.Installer{Script: catsitter.InstallPs1}
	}
	return update.Installer{Script: catsitter.InstallSh}
}
