package tinkerdown

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// SourceConfig represents a data source configuration for lvt-source blocks.
type SourceConfig struct {
	Type        string            `yaml:"type"`                   // exec, pg, rest, csv, json, markdown, sqlite, wasm
	Cmd         string            `yaml:"cmd,omitempty"`          // For exec type
	Query       string            `yaml:"query,omitempty"`        // For pg type
	From        string            `yaml:"from,omitempty"`         // For rest type: API endpoint URL
	File        string            `yaml:"file,omitempty"`         // For csv/json/markdown types
	Anchor      string            `yaml:"anchor,omitempty"`       // For markdown: section anchor (e.g., "#todos")
	DB          string            `yaml:"db,omitempty"`           // For sqlite: database file path
	Table       string            `yaml:"table,omitempty"`        // For sqlite: table name
	Path        string            `yaml:"path,omitempty"`         // For wasm: path to .wasm file
	Headers     map[string]string `yaml:"headers,omitempty"`      // For rest: HTTP headers (env vars expanded)
	QueryParams map[string]string `yaml:"query_params,omitempty"` // For rest: URL query parameters
	ResultPath  string            `yaml:"result_path,omitempty"`  // For rest: dot-path to extract array (e.g., "data.items")
	Readonly    *bool             `yaml:"readonly,omitempty"`     // For markdown/sqlite: read-only mode (default: true)
	Options     map[string]string `yaml:"options,omitempty"`
	Manual      bool              `yaml:"manual,omitempty"`      // For exec: require Run button click
	Format      string            `yaml:"format,omitempty"`      // For exec: output format (json, lines, csv)
	Delimiter   string            `yaml:"delimiter,omitempty"`   // For exec CSV: field delimiter (default ",")
	Env         map[string]string `yaml:"env,omitempty"`         // For exec: environment variables (env vars expanded)
	Timeout     string            `yaml:"timeout,omitempty"`     // For exec/rest: timeout (e.g., "30s", "1m")
}

// StylingConfig represents styling/theme configuration.
type StylingConfig struct {
	Theme        string `yaml:"theme"`
	PrimaryColor string `yaml:"primary_color"`
	Font         string `yaml:"font"`
}

// BlocksConfig represents code block display configuration.
type BlocksConfig struct {
	AutoID          bool   `yaml:"auto_id"`
	IDFormat        string `yaml:"id_format"`
	ShowLineNumbers bool   `yaml:"show_line_numbers"`
}

// FeaturesConfig represents feature flags.
type FeaturesConfig struct {
	HotReload bool `yaml:"hot_reload"`
	Sidebar   bool `yaml:"sidebar"` // Show navigation sidebar
}

// Action defines a custom action that can be triggered via lvt-click.
type Action struct {
	Kind      string              `yaml:"kind"`                // Action kind: "sql", "http", "exec"
	Source    string              `yaml:"source,omitempty"`    // For sql: source name to execute against
	Statement string              `yaml:"statement,omitempty"` // For sql: SQL statement with :param placeholders
	URL       string              `yaml:"url,omitempty"`       // For http: request URL (supports template expressions)
	Method    string              `yaml:"method,omitempty"`    // For http: HTTP method (default: POST)
	Body      string              `yaml:"body,omitempty"`      // For http: request body template
	Cmd       string              `yaml:"cmd,omitempty"`       // For exec: command to run
	Params    map[string]ParamDef `yaml:"params,omitempty"`    // Parameter definitions
	Confirm   string              `yaml:"confirm,omitempty"`   // Confirmation message (triggers dialog)
}

// ParamDef defines a parameter for an action.
type ParamDef struct {
	Type     string `yaml:"type,omitempty"`     // Parameter type: "string", "number", "date", "bool"
	Required bool   `yaml:"required,omitempty"` // Whether the parameter is required
	Default  string `yaml:"default,omitempty"`  // Default value
}

// Frontmatter represents the YAML frontmatter at the top of a markdown file.
type Frontmatter struct {
	// Page metadata
	Title   string      `yaml:"title"`
	Type    string      `yaml:"type"`    // tutorial, guide, reference, playground
	Persist PersistMode `yaml:"persist"` // none, localstorage, server
	Steps   int         `yaml:"steps"`

	// Top-level convenience options
	Sidebar *bool `yaml:"sidebar,omitempty"` // Show navigation sidebar (overrides features.sidebar)

	// Config options (can override livemdtools.yaml)
	Sources  map[string]SourceConfig `yaml:"sources,omitempty"`
	Actions  map[string]Action       `yaml:"actions,omitempty"`
	Styling  *StylingConfig          `yaml:"styling,omitempty"`
	Blocks   *BlocksConfig           `yaml:"blocks,omitempty"`
	Features *FeaturesConfig         `yaml:"features,omitempty"`
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

	// Extract and collect livemdtools code blocks (but don't remove from AST)
	var codeBlocks []*CodeBlock
	blockMap := make(map[ast.Node]*CodeBlock) // Map AST nodes to CodeBlocks
	lineOffset := bytes.Count(content[:len(content)-len(remaining)], []byte("\n"))

	// Walk AST and identify livemdtools code blocks
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
				// This is a livemdtools block - collect it and map it to AST node
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

	// Post-process HTML to add data attributes to livemdtools blocks
	html := htmlBuf.String()
	html = injectBlockAttributes(html, codeBlocks, frontmatter.Sources)

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

// injectBlockAttributes post-processes HTML to wrap livemdtools code blocks with data attributes.
func injectBlockAttributes(html string, blocks []*CodeBlock, sources map[string]SourceConfig) string {
	// For each livemdtools block, find its HTML representation and wrap it
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
				`<div class="tinkerdown-interactive-block" data-tinkerdown-block data-block-id="%s" data-block-type="lvt" data-language="lvt"`,
				escapeHTML(blockID),
			)

			if stateRef, ok := block.Metadata["state"]; ok {
				container += fmt.Sprintf(` data-state-ref="%s"`, escapeHTML(stateRef))
			}

			// Check if this block has an exec source and add toolbar attributes
			if sources != nil {
				sourceName := getLvtSourceFromContent(block.Content)
				if sourceName != "" {
					if srcCfg, ok := sources[sourceName]; ok && srcCfg.Type == "exec" {
						container += ` data-exec-source="true"`
						container += fmt.Sprintf(` data-exec-command="%s"`, escapeHTML(srcCfg.Cmd))
					}
				}
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
			`<div data-tinkerdown-block data-block-id="%s" data-block-type="%s" data-language="%s"`,
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

// getLvtSourceFromContent extracts the lvt-source attribute value from block content.
// Returns empty string if not found.
func getLvtSourceFromContent(content string) string {
	// Look for lvt-source="name" on any element
	sourceRegex := regexp.MustCompile(`lvt-source="([^"]+)"`)
	match := sourceRegex.FindStringSubmatch(content)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return ""
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

// parseCodeBlock parses a fenced code block and extracts livemdtools metadata.
// Code block info string format: "go server readonly id=counter"
func parseCodeBlock(fenced *ast.FencedCodeBlock, source []byte, lineOffset int) (*CodeBlock, error) {
	// Handle fenced code blocks without language info
	if fenced.Info == nil {
		return nil, nil
	}

	info := string(fenced.Info.Text(source))
	parts := strings.Fields(info)

	if len(parts) == 0 {
		// Not a livemdtools code block, skip
		return nil, nil
	}

	language := parts[0]
	remaining := parts[1:]

	// Check for livemdtools block types
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
		// Not a livemdtools block (just regular code block)
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

// partialRegex matches {{partial "filename"}} or {{partial "filename.md"}}
var partialRegex = regexp.MustCompile(`\{\{\s*partial\s+"([^"]+)"\s*\}\}`)

// ProcessPartials recursively processes {{partial "file.md"}} directives in content.
// baseDir is the directory to resolve relative paths from.
// seen tracks already-included files to prevent circular dependencies.
func ProcessPartials(content []byte, baseDir string, seen map[string]bool) ([]byte, error) {
	if seen == nil {
		seen = make(map[string]bool)
	}

	// Find all partial directives
	matches := partialRegex.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	// Process in reverse order to maintain correct positions
	result := make([]byte, len(content))
	copy(result, content)

	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		fullMatchStart := match[0]
		fullMatchEnd := match[1]
		filenameStart := match[2]
		filenameEnd := match[3]

		filename := string(content[filenameStart:filenameEnd])

		// Resolve path relative to baseDir
		partialPath := filepath.Join(baseDir, filename)
		absPath, err := filepath.Abs(partialPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve partial path '%s': %w", filename, err)
		}

		// Check for circular dependency
		if seen[absPath] {
			return nil, fmt.Errorf("circular dependency detected: %s", absPath)
		}
		seen[absPath] = true

		// Read the partial file
		partialContent, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read partial '%s': %w", filename, err)
		}

		// Strip frontmatter from partial (partials don't contribute frontmatter)
		_, partialBody, err := extractFrontmatter(partialContent)
		if err != nil {
			return nil, fmt.Errorf("failed to parse partial '%s': %w", filename, err)
		}

		// Recursively process partials in the included content
		partialDir := filepath.Dir(absPath)
		processedPartial, err := ProcessPartials(partialBody, partialDir, seen)
		if err != nil {
			return nil, fmt.Errorf("error processing partial '%s': %w", filename, err)
		}

		// Replace the directive with the partial content
		result = append(result[:fullMatchStart], append(processedPartial, result[fullMatchEnd:]...)...)
	}

	return result, nil
}

// ParseMarkdownWithPartials parses markdown with partial file support.
// baseDir is used to resolve relative paths in {{partial "file.md"}} directives.
func ParseMarkdownWithPartials(content []byte, baseDir string) (*Frontmatter, []*CodeBlock, string, error) {
	// First, extract frontmatter before processing partials
	frontmatter, remaining, err := extractFrontmatter(content)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Process partials in the remaining content
	processed, err := ProcessPartials(remaining, baseDir, nil)
	if err != nil {
		return nil, nil, "", err
	}

	// Now parse the processed content (without frontmatter since we already extracted it)
	// We need to reconstruct the content for ParseMarkdown or parse directly here

	// Parse markdown with goldmark
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	reader := text.NewReader(processed)
	doc := md.Parser().Parse(reader)

	// Extract and collect livemdtools code blocks
	var codeBlocks []*CodeBlock
	blockMap := make(map[ast.Node]*CodeBlock)
	lineOffset := bytes.Count(content[:len(content)-len(remaining)], []byte("\n"))

	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if fenced, ok := n.(*ast.FencedCodeBlock); ok {
			block, parseErr := parseCodeBlock(fenced, processed, lineOffset)
			if parseErr != nil {
				return ast.WalkStop, parseErr
			}
			if block != nil {
				codeBlocks = append(codeBlocks, block)
				blockMap[n] = block
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to walk AST: %w", err)
	}

	// Generate HTML
	var htmlBuf bytes.Buffer
	if err := md.Renderer().Render(&htmlBuf, processed, doc); err != nil {
		return nil, nil, "", fmt.Errorf("failed to render HTML: %w", err)
	}

	// Post-process HTML to add data attributes
	htmlStr := htmlBuf.String()
	htmlStr = injectBlockAttributes(htmlStr, codeBlocks, frontmatter.Sources)

	return frontmatter, codeBlocks, htmlStr, nil
}
