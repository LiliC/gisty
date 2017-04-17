package main

import (
	"bufio"
	"fmt"
	colour "github.com/fatih/color"
	"github.com/lilic/gisty/gist"
	flag "github.com/spf13/pflag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	githubToken = "GITHUB_TOKEN"
	editor      = "EDITOR"
)

type Options struct {
	Create   bool
	Public   bool
	Anon     bool
	Desc     string
	Content  string
	Filename string
	Show     string
	Edit     string
	List     bool
}

func printGist(gist *gist.Gist) {
	colour.Set(colour.FgYellow)
	fmt.Printf("ID:  %s\n", gist.ID)
	colour.Unset()
	fmt.Print("URL: ")
	colour.Set(colour.Underline)
	fmt.Println(gist.HTMLURL)
	colour.Unset()
	fmt.Printf("Date: %s\n\n", gist.UpdatedAt)
	if gist.Description != "" {
		fmt.Println(gist.Description)
	}
	for filename, _ := range gist.Files {
		fmt.Println(filename)
	}
	fmt.Println()
}

func runCreate(o Options) int {
	var content io.Reader

	// Content from STDIN.
	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if ((info.Mode() & os.ModeCharDevice) != os.ModeCharDevice) && info.Size() > 0 {
		content = bufio.NewReader(os.Stdin)
	}

	// Content from flag.
	if len(o.Content) > 0 {
		content = strings.NewReader(o.Content)
	}
	if content == nil {
		fmt.Println("Please set your content.")
		return 1
	}
	// Create a user gist.
	token := ""
	if !o.Anon {
		token = os.Getenv(githubToken)

		if token == "" {
			fmt.Printf("Please set the ENV variable $%s.\n", githubToken)
			return 1
		}
	}

	c, _ := ioutil.ReadAll(content)
	requestGist := &gist.Gist{
		Public:      o.Public,
		Description: o.Desc,
		Files: map[gist.GistFilename]gist.GistFile{
			gist.GistFilename(o.Filename): gist.GistFile{
				Content: string(c),
			},
		},
	}
	gist, err := gist.Create(token, requestGist)
	if err != nil {
		log.Fatal(err)
	}
	printGist(gist)
	return 0
}

func runShow(o Options) int {
	token := os.Getenv(githubToken)
	if token == "" {
		fmt.Printf("Please set ENV variable $%s.\n", githubToken)
		return 1
	}
	gist, err := gist.Show(token, o.Show)
	if err != nil {
		log.Fatal(err)
	}
	if gist.ID == "" {
		fmt.Println("Wrong ID.")
		return 1
	}
	printGist(gist)
	return 0
}

func runEdit(o Options) int {
	token := os.Getenv(githubToken)
	if token == "" {
		fmt.Printf("Please set ENV variable $%s.\n", githubToken)
		return 1
	}
	e := os.Getenv(editor)
	if e == "" {
		e = "vim"
	}
	var content []byte
	var filename string
	g, err := gist.Show(token, o.Edit)
	if err != nil {
		log.Fatal(err)
	}
	if g.ID == "" {
		fmt.Println("Wrong gist ID / Non existant gist / No writes to W/R")
		return 1
	}
	for f, gf := range g.Files {
		content = []byte(gf.Content)
		filename = string(f)
	}
	tmpFile, err := ioutil.TempFile(os.TempDir(), "gisty")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(e, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	c, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	requestGist := &gist.Gist{
		Public:      o.Public,
		Description: "",
		Files: map[gist.GistFilename]gist.GistFile{
			gist.GistFilename(filename): gist.GistFile{
				Content: string(c),
			},
		},
	}
	g, err = gist.Update(token, o.Edit, requestGist)
	if err != nil {
		log.Fatal(err)
	}
	printGist(g)
	return 0
}

func runList(o Options) int {
	token := os.Getenv(githubToken)
	if token == "" && o.Anon {
		fmt.Printf("Please set ENV variable $%s.\n", githubToken)
		return 1
	}
	gists, err := gist.List(token)
	if err != nil {
		log.Fatal(err)
	}
	for _, g := range gists {
		printGist(g)
	}
	return 0
}

func Main() int {
	options := Options{}
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flags.PrintDefaults()
	}
	flags.BoolVar(&options.Create, "create", false, "create a private gist that will be stored under your profile.")
	flags.BoolVar(&options.Public, "public", false, "create a public gist.")
	flags.BoolVar(&options.Anon, "anon", false, "create an anonymous private gist.")
	flags.StringVar(&options.Desc, "description", "", "specify gist description, if not provided will be left blank.")
	flags.StringVar(&options.Content, "content", "", "specify content of the gist")
	flags.StringVar(&options.Filename, "filename", "file1.txt", "specify name of the file.")
	flags.StringVar(&options.Show, "show", "", "pass a gist ID and it displays a gist.")
	flags.StringVar(&options.Edit, "edit", "", "pass a gist ID to be able to edit your gist.")
	flags.BoolVar(&options.List, "list", false, "lists first 30 of your gists.")
	flags.Parse(os.Args[1:])

	if options.Create {
		return runCreate(options)
	}
	if options.Show != "" {
		return runShow(options)
	}
	if options.Edit != "" {
		return runEdit(options)
	}
	if options.List {
		return runList(options)
	}

	flags.Usage()
	return 1
}

func main() {
	os.Exit(Main())
}
