package scanner

import (
	"path/filepath"
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

// ShouldIncludeFile checks if file matches include patterns
func (f *FileFilter) ShouldIncludeFile(path string) bool {
	if len(f.includeFiles) == 0 {
		return true // No filters means include all
	}

	relPath, err := filepath.Rel(f.projectRoot, path)
	if err != nil {
		relPath = path
	}

	for _, pattern := range f.includeFiles {
		// Support glob patterns including **
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// Support recursive patterns like **/*.go
		if strings.Contains(pattern, "**") {
			// Convert ** to match any path
			globPattern := strings.ReplaceAll(pattern, "**", "*")
			if matched, _ := filepath.Match(globPattern, relPath); matched {
				return true
			}
		}
	}
	return false
}

// ShouldExcludeFile checks if file matches exclude patterns
func (f *FileFilter) ShouldExcludeFile(path string) bool {
	relPath, err := filepath.Rel(f.projectRoot, path)
	if err != nil {
		relPath = path
	}

	for _, pattern := range f.excludeFiles {
		// Support glob patterns
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// Support recursive patterns like **/*.gen.go
		if strings.Contains(pattern, "**") {
			globPattern := strings.ReplaceAll(pattern, "**", "*")
			if matched, _ := filepath.Match(globPattern, relPath); matched {
				return true
			}
		}
	}
	return false
}

// ShouldProcessFile checks if a file should be processed (included and not excluded)
func (f *FileFilter) ShouldProcessFile(path string) bool {
	return f.ShouldIncludeFile(path) && !f.ShouldExcludeFile(path)
}
