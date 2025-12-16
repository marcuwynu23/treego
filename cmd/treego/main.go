package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/alecthomas/kingpin/v2"
	"github.com/marcuwynu23/treego/treego"
)

func main() {
	app := kingpin.New("treego", "Print directory tree and search files").
		Version("v1.0").
		Author("Mark Wayne Menorca")

	app.UsageTemplate(`treego - Print directory tree and search files

	Author: Mark Wayne Menorca
	GitHub: https://github.com/marcuwynu23

	Usage:
	treego <path> [--search <query>] [--regex <pattern>] [--dirs-only] [--version]

	Flags:
	--search, -s       Search string (prints full path)
	--regex, -r        Regex filter
	--dirs-only, -d    Show only directories
	--version          Show version
	`)

	path := app.Arg("path", "root directory to scan").Required().String()
	search := app.Flag("search", "search string (prints full path)").Short('s').String()
	regexStr := app.Flag("regex", "regex filter").Short('r').String()
	dirsOnly := app.Flag("dirs-only", "show only directories").Short('d').Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var re *regexp.Regexp
	if *regexStr != "" {
		var err error
		re, err = regexp.Compile(*regexStr)
		if err != nil {
			fmt.Println("Invalid regex:", err)
			return
		}
	}

	rootPath := filepath.Clean(*path)
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		fmt.Println("Invalid path:", err)
		return
	}

	root := treego.BuildTreeSafe(rootPath)

	if *search != "" {
		treego.SearchDFS(root, *search)
	} else {
		fmt.Println(rootInfo.Name())
		treego.PrintTreeDFS(root, "", re, *dirsOnly)
	}
}

