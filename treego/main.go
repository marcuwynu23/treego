package treego

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"context"
	"golang.org/x/sync/errgroup"

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

type NameMatcher interface {
	MatchString(s string) bool
}

type ExcludeMatcherKind int

const (
	ExcludeExact ExcludeMatcherKind = iota
	ExcludeGlob
	ExcludeRegex
)

// ExcludeMatcher matches against either a base name or full path.
// Supported formats:
// - exact name (e.g. "node_modules")
// - glob (e.g. "*.pem", "dist/*")
// - regex via "re:<expr>" (Go regexp syntax), matched against name and full path
type ExcludeMatcher struct {
	Raw  string
	Kind ExcludeMatcherKind
	Re   *regexp.Regexp
}

func ParseExcludeMatchers(patterns []string) ([]ExcludeMatcher, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	out := make([]ExcludeMatcher, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "re:") {
			expr := strings.TrimSpace(strings.TrimPrefix(p, "re:"))
			re, err := regexp.Compile(expr)
			if err != nil {
				return nil, err
			}
			out = append(out, ExcludeMatcher{Raw: p, Kind: ExcludeRegex, Re: re})
			continue
		}

		if strings.ContainsAny(p, "*?[]") {
			out = append(out, ExcludeMatcher{Raw: p, Kind: ExcludeGlob})
			continue
		}

		out = append(out, ExcludeMatcher{Raw: p, Kind: ExcludeExact})
	}
	return out, nil
}

func (m ExcludeMatcher) matches(name, fullPath string) bool {
	switch m.Kind {
	case ExcludeExact:
		return name == m.Raw || fullPath == m.Raw
	case ExcludeGlob:
		// filepath.Match is OS-specific for path separators; try both name and normalized path.
		if ok, _ := filepath.Match(m.Raw, name); ok {
			return true
		}
		if ok, _ := filepath.Match(m.Raw, filepath.ToSlash(fullPath)); ok {
			return true
		}
		return false
	case ExcludeRegex:
		if m.Re == nil {
			return false
		}
		return m.Re.MatchString(name) || m.Re.MatchString(fullPath)
	default:
		return false
	}
}

func shouldExclude(excludes []ExcludeMatcher, name, fullPath string) bool {
	for _, ex := range excludes {
		if ex.matches(name, fullPath) {
			return true
		}
	}
	return false
}

func BuildTreeSafe(path string) *Node {
	return BuildTreeSafeWithExcludes(path, nil)
}

func BuildTreeSafeWithExcludes(path string, excludes []ExcludeMatcher) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	max := runtime.GOMAXPROCS(0) * 16
	if max < 32 {
		max = 32
	}
	if max > 512 {
		max = 512
	}

	g.SetLimit(max)

	var mu sync.Mutex

	var build func(string) *Node
	build = func(p string) *Node {
		info, err := os.Stat(p)
		if err != nil {
			cancel()
			return nil
		}

		if shouldExclude(excludes, info.Name(), p) {
			return nil
		}

		node := &Node{
			Name:  info.Name(),
			IsDir: info.IsDir(),
			Path:  p,
		}

		if !info.IsDir() {
			return node
		}

		entries, err := os.ReadDir(p)
		if err != nil {
			cancel()
			return nil
		}

		for _, e := range entries {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			name := e.Name()
			childPath := filepath.Join(p, name)

			if shouldExclude(excludes, name, childPath) {
				continue
			}

			if !e.IsDir() {
				mu.Lock()
				node.Children = append(node.Children, &Node{
					Name: name, Path: childPath, IsDir: false,
				})
				mu.Unlock()
				continue
			}

			g.Go(func() error {
				child := build(childPath)
				if child != nil {
					mu.Lock()
					node.Children = append(node.Children, child)
					mu.Unlock()
				}
				return nil
			})
		}

		return node
	}

	root := build(path)

	_ = g.Wait()

	return root
}

func buildTreeSafe(path string, excludes []ExcludeMatcher, sem chan struct{}) *Node {
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

	if shouldExclude(excludes, info.Name(), path) {
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

	// Fast path: process files inline; process directories with bounded parallelism.
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
	)

	for _, e := range entries {
		select {
		case <-abort:
			return nil
		default:
		}

		name := e.Name()
		childPath := filepath.Join(path, name)
		if shouldExclude(excludes, name, childPath) {
			continue
		}

		isDir := e.IsDir()
		if !isDir {
			// Avoid extra syscalls: trust DirEntry for non-dirs.
			mu.Lock()
			node.Children = append(node.Children, &Node{Name: name, IsDir: false, Path: childPath})
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(childPath string) {
			defer wg.Done()

			// Acquire a slot to bound concurrency.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-abort:
				return
			}

			child := buildTreeSafe(childPath, excludes, sem)
			if child == nil {
				return
			}
			mu.Lock()
			node.Children = append(node.Children, child)
			mu.Unlock()
		}(childPath)
	}

	wg.Wait()

	// Stable ordering improves UX and makes output deterministic:
	// directories first, then files; both sorted by name.
	sort.Slice(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir && !b.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

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

func PrintTreeDFS(node *Node, prefix string, relPrefix string, matcher NameMatcher, dirsOnly bool) {
	for i, child := range node.Children {
		if dirsOnly && !child.IsDir {
			continue
		}
		rel := child.Name
		if relPrefix != "" {
			rel = relPrefix + "/" + child.Name
		}
		relOS := filepath.FromSlash(rel)

		matches := true
		if matcher != nil {
			// Match against both name and relative path. This makes negative lookahead
			// exclusions like "^(?!.*bin).*" work for anything under /bin as well.
			matches = matcher.MatchString(child.Name) || matcher.MatchString(rel) || matcher.MatchString(relOS)
		}

		if !matches {
			if child.IsDir {
				var hasMatch bool
				for _, grand := range child.Children {
					grandRel := rel + "/" + grand.Name
					grandRelOS := filepath.FromSlash(grandRel)
					if matcher.MatchString(grand.Name) || matcher.MatchString(grandRel) || matcher.MatchString(grandRelOS) {
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
			PrintTreeDFS(child, nextPrefix, rel, matcher, dirsOnly)
		}
	}
}

// ResetGlobalState resets the global abort channel and once variable for testing
func ResetGlobalState() {
	abort = make(chan struct{})
	once = sync.Once{}
}

