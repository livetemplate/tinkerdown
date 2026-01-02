package tinkerdown

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

// Known limitations documented in golden tests:
// - CodeBlock.Line values are byte offsets, not line numbers (pre-existing parser.go bug)
// - Pandoc-style heading anchors {#id} are not stripped from heading text (goldmark limitation)
// These behaviors are captured in golden files to detect any changes.

// GoldenOutput represents the serializable output for golden file comparison.
type GoldenOutput struct {
	Frontmatter *FrontmatterOutput `json:"frontmatter,omitempty"`
	CodeBlocks  []*CodeBlockOutput `json:"code_blocks,omitempty"`
	HTMLPreview string             `json:"html_preview,omitempty"`
}

// FrontmatterOutput is a serializable version of Frontmatter.
type FrontmatterOutput struct {
	Title   string                       `json:"title,omitempty"`
	Type    string                       `json:"type,omitempty"`
	Persist string                       `json:"persist,omitempty"`
	Steps   int                          `json:"steps,omitempty"`
	Sidebar *bool                        `json:"sidebar,omitempty"`
	Sources map[string]SourceConfigOutput `json:"sources,omitempty"`
}

// SourceConfigOutput is a serializable version of SourceConfig.
type SourceConfigOutput struct {
	Type     string            `json:"type"`
	Cmd      string            `json:"cmd,omitempty"`
	Query    string            `json:"query,omitempty"`
	URL      string            `json:"url,omitempty"`
	File     string            `json:"file,omitempty"`
	Anchor   string            `json:"anchor,omitempty"`
	DB       string            `json:"db,omitempty"`
	Table    string            `json:"table,omitempty"`
	Path     string            `json:"path,omitempty"`
	Readonly *bool             `json:"readonly,omitempty"`
	Manual   bool              `json:"manual,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

// CodeBlockOutput is a serializable version of CodeBlock.
type CodeBlockOutput struct {
	Type     string            `json:"type"`
	Language string            `json:"language,omitempty"`
	Flags    []string          `json:"flags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Content  string            `json:"content"`
	Line     int               `json:"line"`
}

func TestParserGolden(t *testing.T) {
	testCases, err := filepath.Glob("testdata/golden/*.md")
	require.NoError(t, err)

	if len(testCases) == 0 {
		t.Skip("no golden test cases found in testdata/golden/")
	}

	for _, inputPath := range testCases {
		name := strings.TrimSuffix(filepath.Base(inputPath), ".md")
		t.Run(name, func(t *testing.T) {
			runGoldenTest(t, inputPath)
		})
	}
}

func runGoldenTest(t *testing.T, inputPath string) {
	// Read input markdown
	input, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	// Parse the markdown
	frontmatter, codeBlocks, htmlOutput, err := ParseMarkdown(input)
	require.NoError(t, err)

	// Convert to golden output format
	output := convertToGoldenOutput(frontmatter, codeBlocks, htmlOutput)

	// Serialize to JSON
	got, err := json.MarshalIndent(output, "", "  ")
	require.NoError(t, err)

	goldenPath := strings.TrimSuffix(inputPath, ".md") + ".golden.json"

	if *updateGolden {
		err := os.WriteFile(goldenPath, got, 0644)
		require.NoError(t, err)
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	// Read expected output
	want, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("golden file not found: %s (run with -update to create)", goldenPath)
	}
	require.NoError(t, err)

	// Compare
	assert.JSONEq(t, string(want), string(got))
}

func convertToGoldenOutput(fm *Frontmatter, blocks []*CodeBlock, html string) *GoldenOutput {
	output := &GoldenOutput{}

	if fm != nil {
		output.Frontmatter = &FrontmatterOutput{
			Title:   fm.Title,
			Type:    fm.Type,
			Persist: string(fm.Persist),
			Steps:   fm.Steps,
			Sidebar: fm.Sidebar,
		}

		if len(fm.Sources) > 0 {
			output.Frontmatter.Sources = make(map[string]SourceConfigOutput)
			for name, src := range fm.Sources {
				output.Frontmatter.Sources[name] = SourceConfigOutput{
					Type:     src.Type,
					Cmd:      src.Cmd,
					Query:    src.Query,
					URL:      src.URL,
					File:     src.File,
					Anchor:   src.Anchor,
					DB:       src.DB,
					Table:    src.Table,
					Path:     src.Path,
					Readonly: src.Readonly,
					Manual:   src.Manual,
					Options:  src.Options,
				}
			}
		}
	}

	if len(blocks) > 0 {
		output.CodeBlocks = make([]*CodeBlockOutput, len(blocks))
		for i, block := range blocks {
			output.CodeBlocks[i] = &CodeBlockOutput{
				Type:     block.Type,
				Language: block.Language,
				Flags:    block.Flags,
				Metadata: block.Metadata,
				Content:  block.Content,
				Line:     block.Line,
			}
		}
	}

	// Truncate HTML preview to first 500 chars for readability
	if len(html) > 500 {
		output.HTMLPreview = html[:500] + "..."
	} else if html != "" {
		output.HTMLPreview = html
	}

	return output
}
