package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/kdar/gocurse"
	"log"
	"os"
	"path/filepath"
)

type options struct {
	ApiKey        string `short:"a" long:"api-key" description:"Your CurseForge API key" required:"true"`
	Game          string `short:"g" long:"game" description:"The game in lower case (e.g. wow, war, rom)" required:"true"`
	Slug          string `short:"s" long:"slug" description:"Your addon's slug name" required:"true"`
	Release       bool   `long:"release" description:"Whether the file type is: release" default:"true"`
	Beta          bool   `long:"beta" description:"Whether the file type is: beta"`
	Alpha         bool   `long:"alpha" description:"Whether the file type is: alpha"`
	ShowChangelog bool   `long:"show-changelog" description:"Display the changelog and exit"`
}

func main() {
	log.SetFlags(log.Lshortfile)

	var opts options
	var parser = flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		os.Exit(-1)
	}

	changelog, err := gitChangelog()
	if err != nil {
		log.Fatal(err)
	}

	if opts.ShowChangelog {
		fmt.Println(changelog)
		os.Exit(0)
	}

	// meta, err := gocurse.GetPkgMeta()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	if !isGitRepo() {
		log.Fatal("Not in a git repository.")
	}

	latestTag, err := gitLatestTag()
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	name := fmt.Sprintf("%s %s", filepath.Base(dir), latestTag)

	client := gocurse.New(opts.Game, opts.ApiKey)

	filetype := ""
	switch {
	case opts.Beta:
		filetype = gocurse.FileTypeBeta
	case opts.Alpha:
		filetype = gocurse.FileTypeAlpha
	default:
		filetype = gocurse.FileTypeRelease
	}

	fileopt := &gocurse.FileOptions{
		Name:              name,
		FileType:          filetype,
		ChangeLog:         changelog,
		ChangeMarkupType:  gocurse.MarkupTypePlain,
		KnownCaveats:      "",
		CaveatsMarkupType: gocurse.MarkupTypePlain,
	}

	archive, err := gitArchive(filepath.Base(dir) + "/")
	if err != nil {
		log.Fatal(err)
	}

	err = client.UploadFile(fileopt, opts.Slug, name+".zip", archive)
	if err != nil {
		log.Fatal(err)
	}
}
