package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/alecthomas/kingpin/v2"
)

type Node struct {
	Name     string
	Children []*Node
	IsDir    bool
	Path     string
}

type job struct {
	path string
	node *Node
}

var abort = make(chan struct{}) // closed to abort all goroutines

func buildTreeSafe(path string) *Node {
	select {
	case <-abort:
		// someone already triggered abort, stop immediately
		return nil
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		closeOnce()
		return nil
	}

	node := &Node{Name: info.Name(), IsDir: info.IsDir(), Path: path}
	if !info.IsDir() {
		return node
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		closeOnce()
		return nil
	}

	var wg sync.WaitGroup
	childNodes := make([]*Node, len(entries))

	for i, e := range entries {
		wg.Add(1)
		go func(i int, e os.DirEntry) {
			defer wg.Done()
			select {
			case <-abort:
				return
			default:
			}
			childPath := filepath.Join(path, e.Name())
			childNodes[i] = buildTreeSafe(childPath)
		}(i, e)
	}

	wg.Wait()

	for _, c := range childNodes {
		if c != nil {
			node.Children = append(node.Children, c)
		}
	}

	return node
}

// helper to close abort channel only once
var once sync.Once
func closeOnce() {
	once.Do(func() {
		close(abort)
	})
}



func searchDFS(node *Node, query string) {
	if strings.Contains(strings.ToLower(node.Name), strings.ToLower(query)) {
		fmt.Println(node.Path)
	}
	for _, child := range node.Children {
		searchDFS(child, query)
	}
}

func printTreeDFS(node *Node, prefix string, regex *regexp.Regexp, dirsOnly bool) {
	for i, child := range node.Children {
		if dirsOnly && !child.IsDir {
			continue
		}
		if regex != nil && !regex.MatchString(child.Name) {
			if child.IsDir {
				var hasMatch bool
				for _, grand := range child.Children {
					if regex.MatchString(grand.Name) {
						hasMatch = true
						break
					}
				}
				if !hasMatch {
					continue
				}
			} else {
				continue
			}
		}
		last := i == len(node.Children)-1
		branch := "├── "
		nextPrefix := prefix + "│   "
		if last {
			branch = "└── "
			nextPrefix = prefix + "    "
		}
		fmt.Println(prefix + branch + child.Name)
		if child.IsDir {
			printTreeDFS(child, nextPrefix, regex, dirsOnly)
		}
	}
}

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

	root := buildTreeSafe(rootPath)

	if *search != "" {
		searchDFS(root, *search)
	} else {
		fmt.Println(rootInfo.Name())
		printTreeDFS(root, "", re, *dirsOnly)
	}
}
