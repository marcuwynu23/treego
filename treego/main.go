package treego

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type Node struct {
	Name     string
	Children []*Node
	IsDir    bool
	Path     string
}

var abort = make(chan struct{}) // optional global abort

type NameMatcher interface {
	MatchString(s string) bool
}

type ExcludeMatcherKind int

const (
	ExcludeExact ExcludeMatcherKind = iota
	ExcludeGlob
	ExcludeRegex
)

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
			expr := strings.TrimPrefix(p, "re:")
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
	}

	return false
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

func BuildTreeSafeWithExcludes(rootPath string, excludes []ExcludeMatcher) *Node {
	root := &Node{
		Name:  filepath.Base(rootPath),
		Path:  rootPath,
		IsDir: true,
	}

	type task struct {
		path   string
		parent *Node
	}

	queue := []task{{path: rootPath, parent: root}}
	var mu sync.Mutex

	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]

		info, err := os.Stat(t.path)
		if err != nil {
			continue
		}

		if shouldExclude(excludes, info.Name(), t.path) {
			continue
		}

		node := &Node{
			Name:  info.Name(),
			Path:  t.path,
			IsDir: info.IsDir(),
		}

		if t.parent != nil {
			mu.Lock()
			t.parent.Children = append(t.parent.Children, node)
			mu.Unlock()
		}

		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(t.path)
		if err != nil {
			continue
		}

		for _, e := range entries {
			queue = append(queue, task{
				path:   filepath.Join(t.path, e.Name()),
				parent: node,
			})
		}
	}

	return root
}

// ---------- SORTING LOGIC (FIXED) ----------

func sortChildren(children []*Node) {
	sort.Slice(children, func(i, j int) bool {
		a, b := children[i], children[j]

		aHidden := strings.HasPrefix(a.Name, ".")
		bHidden := strings.HasPrefix(b.Name, ".")

		// 1. directories first
		if a.IsDir != b.IsDir {
			return a.IsDir && !b.IsDir
		}

		// 2. hidden files/folders go last inside same group
		if aHidden != bHidden {
			return !aHidden && bHidden
		}

		// 3. alphabetical
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})

	// recursively sort children
	for _, c := range children {
		if c.IsDir && len(c.Children) > 0 {
			sortChildren(c.Children)
		}
	}
}

// ---------- PRINT ----------

func PrintTreeDFS(node *Node, prefix string, relPrefix string, matcher NameMatcher, dirsOnly bool) {
	sortChildren(node.Children)

	for i, child := range node.Children {
		if dirsOnly && !child.IsDir {
			continue
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
			PrintTreeDFS(child, nextPrefix, "", matcher, dirsOnly)
		}
	}
}

// ---------- OPTIONAL SEARCH ----------

func SearchDFS(node *Node, query string) {
	if strings.Contains(strings.ToLower(node.Name), strings.ToLower(query)) {
		fmt.Println(node.Path)
	}
	for _, c := range node.Children {
		SearchDFS(c, query)
	}
}

// ---------- RESET ----------

func ResetGlobalState() {
	abort = make(chan struct{})
}