package treego

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

func BuildTreeSafe(path string) *Node {
	select {
	case <-abort:
		// someone already triggered abort, stop immediately
		return nil
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		CloseOnce()
		return nil
	}

	node := &Node{Name: info.Name(), IsDir: info.IsDir(), Path: path}
	if !info.IsDir() {
		return node
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		CloseOnce()
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
			childNodes[i] = BuildTreeSafe(childPath)
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
func CloseOnce() {
	once.Do(func() {
		close(abort)
	})
}



func SearchDFS(node *Node, query string) {
	if strings.Contains(strings.ToLower(node.Name), strings.ToLower(query)) {
		fmt.Println(node.Path)
	}
	for _, child := range node.Children {
		SearchDFS(child, query)
	}
}

func PrintTreeDFS(node *Node, prefix string, regex *regexp.Regexp, dirsOnly bool) {
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
			PrintTreeDFS(child, nextPrefix, regex, dirsOnly)
		}
	}
}

// ResetGlobalState resets the global abort channel and once variable for testing
func ResetGlobalState() {
	abort = make(chan struct{})
	once = sync.Once{}
}

