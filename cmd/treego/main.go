package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/alecthomas/kingpin/v2"
	"github.com/dlclark/regexp2"
	"github.com/marcuwynu23/treego/treego"
)

type goRegexpMatcher struct{ re *regexp.Regexp }

func (m goRegexpMatcher) MatchString(s string) bool {
	return m.re != nil && m.re.MatchString(s)
}

type perlRegexpMatcher struct{ re *regexp2.Regexp }

func (m perlRegexpMatcher) MatchString(s string) bool {
	if m.re == nil {
		return false
	}
	ok, err := m.re.MatchString(s)
	return err == nil && ok
}

func main() {
	app := kingpin.New("treego", "Print directory tree and search files").
		Version("v1.0").
		Author("Mark Wayne Menorca")

	app.UsageTemplate(`treego - Print directory tree and search files

	Author: Mark Wayne Menorca
	GitHub: https://github.com/marcuwynu23

	Usage:
	treego <path> [--search <query>] [--regex <pattern>] [--exclude <pattern>...] [--dirs-only] [--version]

	Flags:
	--search, -s       Search string (prints full path)
	--regex, -r        Regex filter
	--exclude, -x      Exclude pattern (repeatable). Supports exact name (node_modules), glob (*.pem), or regex (re:<expr>)
	--dirs-only, -d    Show only directories
	--version          Show version
	`)

	path := app.Arg("path", "root directory to scan").Required().String()
	search := app.Flag("search", "search string (prints full path)").Short('s').String()
	regexStr := app.Flag("regex", "regex filter").Short('r').String()
	excludePatterns := app.Flag("exclude", "exclude pattern (repeatable). supports exact name, glob, or regex re:<expr>").Short('x').Strings()
	dirsOnly := app.Flag("dirs-only", "show only directories").Short('d').Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	var matcher treego.NameMatcher
	if *regexStr != "" {
		// Prefer Go regexp for speed when possible, but fallback to a Perl-like engine
		// to support lookaheads such as (?!...).
		if re, goErr := regexp.Compile(*regexStr); goErr == nil {
			matcher = goRegexpMatcher{re: re}
		} else {
			perl, err2 := regexp2.Compile(*regexStr, 0)
			if err2 != nil {
				fmt.Println("Invalid regex:", goErr)
				return
			}
			perl.MatchTimeout = regexp2.DefaultMatchTimeout
			// If user anchored to the whole name (e.g. ^...$), keep behavior.
			// Otherwise behavior stays "match anywhere" like Go's regexp does.
			matcher = perlRegexpMatcher{re: perl}
		}
	}

	excludes, err := treego.ParseExcludeMatchers(*excludePatterns)
	if err != nil {
		fmt.Println("Invalid exclude pattern:", err)
		return
	}

	rootPath := filepath.Clean(*path)
	rootInfo, err := os.Stat(rootPath)
	if err != nil {
		fmt.Println("Invalid path:", err)
		return
	}

	root := treego.BuildTreeSafeWithExcludes(rootPath, excludes)
	if root == nil {
		// Either excluded or an error occurred during traversal.
		return
	}

	if *search != "" {
		treego.SearchDFS(root, *search)
	} else {
		fmt.Println(rootInfo.Name())
		// Make regex match against names (like before).
		// Users who want to match paths should use --exclude re:<expr>.
		treego.PrintTreeDFS(root, "", "", matcher, *dirsOnly)
	}
}

