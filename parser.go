package livepage

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Frontmatter represents the YAML frontmatter at the top of a markdown file.
type Frontmatter struct {
	Title   string      `yaml:"title"`
	Type    string      `yaml:"type"`    // tutorial, guide, reference, playground
	Persist PersistMode `yaml:"persist"` // none, localstorage, server
	Steps   int         `yaml:"steps"`
}

// CodeBlock represents a code block extracted from markdown.
type CodeBlock struct {
	Type     string            // "server", "wasm", "lvt"
	Language string            // "go", etc.
	Flags    []string          // "readonly", "editable"
	Metadata map[string]string // id, state, etc.
	Content  string
	Line     int // Line number in source file
}

// ParseMarkdown parses a markdown file and extracts frontmatter and code blocks.
func ParseMarkdown(content []byte) (*Frontmatter, []*CodeBlock, string, error) {
	// Extract frontmatter
	frontmatter, remaining, err := extractFrontmatter(content)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse markdown with goldmark
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	reader := text.NewReader(remaining)
	doc := md.Parser().Parse(reader)

	// Extract code blocks and remove livepage blocks from AST
	var codeBlocks []*CodeBlock
	lineOffset := bytes.Count(content[:len(content)-len(remaining)], []byte("\n"))

	// First pass: identify and collect livepage code blocks
	var toRemove []ast.Node
	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if fenced, ok := n.(*ast.FencedCodeBlock); ok {
			block, parseErr := parseCodeBlock(fenced, remaining, lineOffset)
			if parseErr != nil {
				return ast.WalkStop, parseErr
			}
			if block != nil {
				// This is a livepage block - collect it and mark for removal
				codeBlocks = append(codeBlocks, block)
				toRemove = append(toRemove, n)
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to walk AST: %w", err)
	}

	// Second pass: remove livepage code blocks from AST
	for _, node := range toRemove {
		parent := node.Parent()
		if parent != nil {
			parent.RemoveChild(parent, node)
		}
	}

	// Generate static HTML for prose (livepage code blocks removed)
	var htmlBuf bytes.Buffer
	if err := md.Renderer().Render(&htmlBuf, remaining, doc); err != nil {
		return nil, nil, "", fmt.Errorf("failed to render HTML: %w", err)
	}

	return frontmatter, codeBlocks, htmlBuf.String(), nil
}

// extractFrontmatter extracts YAML frontmatter from the beginning of content.
// Returns the parsed frontmatter and the remaining content.
func extractFrontmatter(content []byte) (*Frontmatter, []byte, error) {
	if !bytes.HasPrefix(content, []byte("---\n")) {
		// No frontmatter, use defaults
		return &Frontmatter{
			Type:    "tutorial",
			Persist: PersistLocalStorage,
		}, content, nil
	}

	// Find the closing ---
	endIdx := bytes.Index(content[4:], []byte("\n---\n"))
	if endIdx == -1 {
		return nil, nil, fmt.Errorf("unclosed frontmatter")
	}

	yamlContent := content[4 : 4+endIdx]
	remaining := content[4+endIdx+5:] // Skip "\n---\n"

	var fm Frontmatter
	if err := yaml.Unmarshal(yamlContent, &fm); err != nil {
		return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults
	if fm.Type == "" {
		fm.Type = "tutorial"
	}
	if fm.Persist == "" {
		fm.Persist = PersistLocalStorage
	}

	return &fm, remaining, nil
}

// parseCodeBlock parses a fenced code block and extracts livepage metadata.
// Code block info string format: "go server readonly id=counter"
func parseCodeBlock(fenced *ast.FencedCodeBlock, source []byte, lineOffset int) (*CodeBlock, error) {
	info := string(fenced.Info.Text(source))
	parts := strings.Fields(info)

	if len(parts) == 0 {
		// Not a livepage code block, skip
		return nil, nil
	}

	language := parts[0]
	remaining := parts[1:]

	// Check for livepage block types
	blockType := ""
	flags := []string{}
	metadata := make(map[string]string)

	// Special case: if language is "lvt", it's the block type
	if language == "lvt" {
		blockType = "lvt"
	}

	for _, part := range remaining {
		if strings.Contains(part, "=") {
			// Metadata: key=value
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				metadata[kv[0]] = strings.Trim(kv[1], `"'`)
			}
		} else {
			// Block type or flag
			switch part {
			case "server", "wasm", "lvt":
				blockType = part
			case "readonly", "editable":
				flags = append(flags, part)
			case "interactive":
				// Interactive is a special flag for lvt blocks
				flags = append(flags, part)
			default:
				// Unknown flag, ignore
			}
		}
	}

	if blockType == "" {
		// Not a livepage block (just regular code block)
		return nil, nil
	}

	// Extract content
	var buf bytes.Buffer
	lines := fenced.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		buf.Write(line.Value(source))
	}

	return &CodeBlock{
		Type:     blockType,
		Language: language,
		Flags:    flags,
		Metadata: metadata,
		Content:  buf.String(),
		Line:     lineOffset + fenced.Lines().At(0).Start,
	}, nil
}

// ParseFiles parses markdown files and updates the Page.
func (p *Page) ParseFiles(files ...string) error {
	// TODO: Implement multi-file parsing
	// For now, assume single file
	return fmt.Errorf("not implemented")
}
