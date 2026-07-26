package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// quickFindNoisy is the set of directory names we never descend into.
var quickFindNoisy = map[string]bool{
	// JS / Python
	"node_modules": true, "__pycache__": true,
	// VCS
	".git": true, ".svn": true, ".hg": true,
	// Go module cache (pkg/mod can be enormous)
	"pkg": true, "mod": true, "go-workspace": true,
	// Common build outputs / caches
	"vendor": true, ".cache": true, "dist": true, "build": true, "target": true,
	// Editors / Temp / Assets
	".vscode": true, ".idea": true, "tmp": true, "temp": true,
}

// isDirOrSymlinkToDir returns true for real directories and symlinks to directories.
// e.IsDir() returns false for symlinks even when they point to a dir, so we
// fall back to os.Stat (which follows symlinks) when the entry is a symlink.
// ponytail: one Stat call per symlink entry; ceiling: EvalSymlinks for cycle detection.
func isDirOrSymlinkToDir(e os.DirEntry, fullPath string) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	fi, err := os.Stat(fullPath) // follows the symlink
	return err == nil && fi.IsDir()
}

// quickFindRoots returns deduped walk roots: home, /data (if present), and a
// top-level ancestor of currentDir when it lives outside home.
// ponytail: /data hardcoded for this machine; ceiling: user-configurable roots.
func quickFindRoots(home, currentDir string) []string {
	out := []string{home}
	// Always probe /data – common external data partition on this machine.
	if fi, err := os.Stat("/data"); err == nil && fi.IsDir() {
		out = append(out, "/data")
	}
	// If currentDir is inside home or already covered, skip extra climb.
	if strings.HasPrefix(currentDir, home+string(filepath.Separator)) || currentDir == home {
		return out
	}
	// Climb toward filesystem root, stop one level below "/".
	dir := currentDir
	for {
		parent := filepath.Dir(dir)
		if parent == dir || parent == "/" {
			break
		}
		dir = parent
	}
	// Add dir only if it's not already in out.
	if dir != home && dir != "/" {
		for _, r := range out {
			if r == dir {
				return out
			}
		}
		out = append(out, dir)
	}
	return out
}

// buildQuickFindResults returns the top-10 best-matching directories for query.
// Bookmarks first, then walks home + currentDir ancestor up to depth 6.
// Matches on basename only; ranked: exact(3) > prefix(2) > contains(1).
// ponytail: sync walk, depth 6, fixed roots; ceiling: async walk + pre-built index.
func (m model) buildQuickFindResults(query string) []ShortcutEntry {
	q := strings.ToLower(query)

	// 1. Bookmarks first – match basename
	var results []ShortcutEntry
	for _, b := range m.bookmarks {
		if q == "" || strings.Contains(strings.ToLower(b.Name), q) {
			results = append(results, b)
		}
	}
	if q == "" {
		return results
	}

	// 2. Walk roots, collect all basename matches with a relevance score.
	home, _ := os.UserHomeDir()
	roots := quickFindRoots(home, m.tabs[m.activeTab].currentDir)
	seen := make(map[string]bool, len(results))
	for _, r := range results {
		seen[r.Path] = true
	}

	type scored struct {
		entry ShortcutEntry
		score int
	}
	var candidates []scored

	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 6 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			name := e.Name()
			full := filepath.Join(dir, name)
			isDir := isDirOrSymlinkToDir(e, full)

			if isDir {
				// Skip hidden directories to protect search performance
				if strings.HasPrefix(name, ".") || quickFindNoisy[name] {
					continue
				}
				walk(full, depth+1)
			} else {
				// For files, skip noisy ones. Skip hidden files unless they are whitelisted.
				if quickFindNoisy[name] {
					continue
				}
				if strings.HasPrefix(name, ".") {
					allowed := false
					for _, allowedFile := range m.config.AllowedHiddenFiles {
						if name == allowedFile {
							allowed = true
							break
						}
					}
					if !allowed {
						continue
					}
				}
			}

			lname := strings.ToLower(name)
			score := 0
			switch {
			case lname == q:
				score = 3
			case strings.HasPrefix(lname, q):
				score = 2
			case strings.Contains(lname, q):
				score = 1
			}
			if score > 0 && !seen[full] {
				icon := FileIcon
				if isDir {
					icon = DirIcon
				}
				candidates = append(candidates, scored{
					entry: ShortcutEntry{Name: name, Path: full, Icon: icon},
					score: score,
				})
				seen[full] = true
			}
		}
	}

	for _, root := range roots {
		walk(root, 0)
	}

	// 3. Sort by score descending; stable preserves walk order within same score.
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// 4. Append top 10 total (bookmarks already in results).
	const maxResults = 10
	for _, c := range candidates {
		if len(results) >= maxResults {
			break
		}
		results = append(results, c.entry)
	}
	return results
}
