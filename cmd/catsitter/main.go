package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/xanderwasserman/catsitter/internal/app"
	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/host"
	"github.com/xanderwasserman/catsitter/internal/pet"
	"github.com/xanderwasserman/catsitter/internal/pet/cat"
	"github.com/xanderwasserman/catsitter/internal/tray"
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
	session := app.New(live, saved, path, backends)
	session.Start()
	defer session.Shutdown()
	if err := pet.Run(cat.New(), session, tray.New(session), backends.Pump, backends.Overlay); err != nil {
		log.Fatal(err)
	}
}
