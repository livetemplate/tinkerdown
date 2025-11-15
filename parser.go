package livepage

import (
	"bytes"
	"fmt"
	"html"
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

	// Extract and collect livepage code blocks (but don't remove from AST)
	var codeBlocks []*CodeBlock
	blockMap := make(map[ast.Node]*CodeBlock) // Map AST nodes to CodeBlocks
	lineOffset := bytes.Count(content[:len(content)-len(remaining)], []byte("\n"))

	// Walk AST and identify livepage code blocks
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
				// This is a livepage block - collect it and map it to AST node
				codeBlocks = append(codeBlocks, block)
				blockMap[n] = block
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to walk AST: %w", err)
	}

	// Generate HTML (basic rendering first)
	var htmlBuf bytes.Buffer
	if err := md.Renderer().Render(&htmlBuf, remaining, doc); err != nil {
		return nil, nil, "", fmt.Errorf("failed to render HTML: %w", err)
	}

	// Post-process HTML to add data attributes to livepage blocks
	html := htmlBuf.String()
	html = injectBlockAttributes(html, codeBlocks)

	return frontmatter, codeBlocks, html, nil
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

// injectBlockAttributes post-processes HTML to wrap livepage code blocks with data attributes.
func injectBlockAttributes(html string, blocks []*CodeBlock) string {
	// For each livepage block, find its HTML representation and wrap it
	for i, block := range blocks {
		// Determine readonly/editable
		readonly := containsFlag(block.Flags, "readonly")
		editable := containsFlag(block.Flags, "editable")
		if !readonly && !editable {
			if block.Type == "server" {
				readonly = true
			} else if block.Type == "wasm" {
				editable = true
			}
		}

		// Get block ID (use same logic as page.go getBlockID)
		blockID := block.Metadata["id"]
		if blockID == "" {
			// Auto-generate: type-index (e.g., "server-0", "lvt-1")
			// Must match the index used in buildBlocks()
			blockID = fmt.Sprintf("%s-%d", block.Type, i)
		}

		// For interactive (lvt) blocks, replace with a container div instead of code block
		if block.Type == "lvt" {
			// Build container div with data attributes
			container := fmt.Sprintf(
				`<div class="livepage-interactive-block" data-livepage-block data-block-id="%s" data-block-type="lvt" data-language="lvt"`,
				escapeHTML(blockID),
			)

			if stateRef, ok := block.Metadata["state"]; ok {
				container += fmt.Sprintf(` data-state-ref="%s"`, escapeHTML(stateRef))
			}

			// Add a placeholder that will be replaced by WebSocket initial state
			container += ` data-interactive-content><div class="loading">Connecting...</div></div>`

			// Find and replace the <pre><code> block with our container
			oldPre := fmt.Sprintf(`<pre><code class="language-%s">`, block.Language)

			// Find the closing tags
			preStart := strings.Index(html, oldPre)
			if preStart != -1 {
				// Find the end of this code block
				codeEnd := strings.Index(html[preStart:], "</code></pre>")
				if codeEnd != -1 {
					// Replace the entire <pre><code>...</code></pre> with our container
					before := html[:preStart]
					after := html[preStart+codeEnd+len("</code></pre>"):]
					html = before + container + after
				}
			}
			continue
		}

		// For server/wasm blocks, wrap the existing <pre><code> with attributes
		wrapper := fmt.Sprintf(
			`<div data-livepage-block data-block-id="%s" data-block-type="%s" data-language="%s"`,
			escapeHTML(blockID),
			escapeHTML(block.Type),
			escapeHTML(block.Language),
		)

		if readonly {
			wrapper += ` data-readonly="true"`
		}
		if editable {
			wrapper += ` data-editable="true"`
		}
		wrapper += ">"

		// Find <pre><code> blocks and wrap the first match
		oldPre := fmt.Sprintf(`<pre><code class="language-%s">`, block.Language)
		newPre := wrapper + oldPre

		// Only replace the first occurrence (to handle multiple blocks)
		html = strings.Replace(html, oldPre, newPre, 1)

		// Close the wrapper after </pre>
		html = strings.Replace(html, "</pre>", "</pre></div>", 1)
	}

	return html
}

// containsFlag checks if a flag is in the flags slice.
func containsFlag(flags []string, flag string) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}
	return false
}

// escapeHTML escapes HTML special characters.
func escapeHTML(s string) string {
	return html.EscapeString(s)
}

// parseCodeBlock parses a fenced code block and extracts livepage metadata.
// Code block info string format: "go server readonly id=counter"
func parseCodeBlock(fenced *ast.FencedCodeBlock, source []byte, lineOffset int) (*CodeBlock, error) {
	// Handle fenced code blocks without language info
	if fenced.Info == nil {
		return nil, nil
	}

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
