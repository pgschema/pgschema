package include

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Processor handles processing SQL files with \i include directives
type Processor struct {
	baseDir string
	visited map[string]bool
}

// NewProcessor creates a new include processor for the given base directory
func NewProcessor(baseDir string) *Processor {
	return &Processor{
		baseDir: baseDir,
		visited: make(map[string]bool),
	}
}

// ProcessFile processes a SQL file and resolves all \i include directives
func (p *Processor) ProcessFile(filename string) (string, error) {
	// Reset visited map for each top-level file processing
	p.visited = make(map[string]bool)
	
	// Get absolute path to ensure consistent path handling
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", filename, err)
	}
	
	// Update base directory based on the input file's directory
	p.baseDir = filepath.Dir(absPath)
	
	return p.processFileRecursive(absPath)
}

// processFileRecursive recursively processes a file and its includes
func (p *Processor) processFileRecursive(filename string) (string, error) {
	// Check for circular dependencies
	if p.visited[filename] {
		return "", fmt.Errorf("circular dependency detected: %s", filename)
	}
	
	// Mark file as visited
	p.visited[filename] = true
	defer func() {
		// Unmark after processing to allow the same file to be included in different branches
		delete(p.visited, filename)
	}()
	
	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	
	// Process includes in the current file
	currentDir := filepath.Dir(filename)
	processedContent, err := p.processIncludes(string(content), currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to process includes in %s: %w", filename, err)
	}
	
	return processedContent, nil
}

// processIncludes processes \i directives in the given content
func (p *Processor) processIncludes(content string, currentDir string) (string, error) {
	// Regex to match \i directives
	// Matches: \i filename or \i filename; (with optional semicolon)
	includeRegex := regexp.MustCompile(`^\s*\\i\s+([^\s;]+)\s*;?\s*$`)
	
	lines := strings.Split(content, "\n")
	var result strings.Builder
	
	for i, line := range lines {
		matches := includeRegex.FindStringSubmatch(line)
		if matches != nil {
			// Found an include directive
			includePath := matches[1]
			
			// Resolve the include path
			resolvedPath, err := p.resolveIncludePath(includePath, currentDir)
			if err != nil {
				return "", fmt.Errorf("line %d: failed to resolve include path %s: %w", i+1, includePath, err)
			}
			
			// Process the included file recursively
			includedContent, err := p.processFileRecursive(resolvedPath)
			if err != nil {
				return "", fmt.Errorf("line %d: failed to process included file %s: %w", i+1, resolvedPath, err)
			}
			
			// Add the included content (with a newline to separate from surrounding content)
			result.WriteString(includedContent)
			if !strings.HasSuffix(includedContent, "\n") {
				result.WriteString("\n")
			}
		} else {
			// Regular line, add as-is
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
		}
	}
	
	return result.String(), nil
}

// resolveIncludePath resolves an include path relative to the current directory
// Only allows files within the base directory and its subdirectories
func (p *Processor) resolveIncludePath(includePath string, currentDir string) (string, error) {
	// Clean the path to remove any . or .. components
	cleanPath := filepath.Clean(includePath)
	
	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("directory traversal not allowed: %s", includePath)
	}
	
	// Resolve relative to current directory
	resolvedPath := filepath.Join(currentDir, cleanPath)
	
	// Get absolute path
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// Ensure the resolved path is within the base directory
	baseAbs, err := filepath.Abs(p.baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}
	
	// Check if the resolved path is within the base directory
	relPath, err := filepath.Rel(baseAbs, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("include path %s is outside the base directory %s", includePath, p.baseDir)
	}
	
	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("included file does not exist: %s", absPath)
	}
	
	return absPath, nil
}