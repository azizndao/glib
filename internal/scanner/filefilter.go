package scanner

import (
	"path/filepath"
	"regexp"
	"strings"
)

// FileFilter provides file and directory filtering functionality
type FileFilter struct {
	projectRoot  string
	excludeDirs  []string
	includeFiles []string
	excludeFiles []string
}

// NewFileFilter creates a new file filter
func NewFileFilter(projectRoot string, excludeDirs, includeFiles, excludeFiles []string) *FileFilter {
	return &FileFilter{
		projectRoot:  projectRoot,
		excludeDirs:  excludeDirs,
		includeFiles: includeFiles,
		excludeFiles: excludeFiles,
	}
}

// ShouldExcludeDir checks if a directory should be excluded from scanning
func (f *FileFilter) ShouldExcludeDir(path string) bool {
	// Get relative path from project root to avoid matching system paths like /tmp/
	relPath, err := filepath.Rel(f.projectRoot, path)
	if err != nil {
		relPath = path
	}

	// Check if this is the project root
	if relPath == "." {
		return false
	}

	dirName := filepath.Base(path)
	for _, pattern := range f.excludeDirs {
		// Support both simple names and glob patterns
		if matched, _ := filepath.Match(pattern, dirName); matched {
			return true
		}
		// Check if any component of relative path matches pattern
		parts := strings.SplitSeq(relPath, string(filepath.Separator))
		for part := range parts {
			if matched, _ := filepath.Match(pattern, part); matched {
				return true
			}
		}
	}
	return false
}

func matchesGlobPattern(pattern, candidate string) bool {
	if pattern == "" {
		return candidate == ""
	}

	pattern = filepath.ToSlash(pattern)
	candidate = filepath.ToSlash(candidate)
	pattern = strings.TrimPrefix(pattern, "./")
	candidate = strings.TrimPrefix(candidate, "./")

	if pattern == candidate {
		return true
	}

	if !strings.ContainsAny(pattern, "*?[") {
		return false
	}

	var regexPattern strings.Builder
	regexPattern.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch {
		case pattern[i] == '*' && i+1 < len(pattern) && pattern[i+1] == '*':
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				regexPattern.WriteString("(?:.*/)?")
				i += 2
				continue
			}
			regexPattern.WriteString(".*")
			i++
		case pattern[i] == '*':
			regexPattern.WriteString("[^/]*")
		case pattern[i] == '?':
			regexPattern.WriteString("[^/]")
		default:
			switch pattern[i] {
			case '.', '+', '(', ')', '^', '$', '{', '}', '[', ']', '|', '\\':
				regexPattern.WriteString("\\")
			}
			regexPattern.WriteString(string(pattern[i]))
		}
	}
	regexPattern.WriteString("$")

	matched, err := regexp.MatchString(regexPattern.String(), candidate)
	return err == nil && matched
}

func matchesAnyPattern(projectRoot, path string, patterns []string) bool {
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		relPath = path
	}

	baseName := filepath.Base(path)
	for _, pattern := range patterns {
		if matchesGlobPattern(pattern, baseName) || matchesGlobPattern(pattern, relPath) {
			return true
		}
	}
	return false
}

// ShouldIncludeFile checks if file matches include patterns
func (f *FileFilter) ShouldIncludeFile(path string) bool {
	if len(f.includeFiles) == 0 {
		return true // No filters means include all
	}

	return matchesAnyPattern(f.projectRoot, path, f.includeFiles)
}

// ShouldExcludeFile checks if file matches exclude patterns
func (f *FileFilter) ShouldExcludeFile(path string) bool {
	return matchesAnyPattern(f.projectRoot, path, f.excludeFiles)
}

// ShouldProcessFile checks if a file should be processed (included and not excluded)
func (f *FileFilter) ShouldProcessFile(path string) bool {
	return f.ShouldIncludeFile(path) && !f.ShouldExcludeFile(path)
}
