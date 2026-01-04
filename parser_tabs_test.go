package tinkerdown

import (
	"strings"
	"testing"
)

func TestParseTabsFromContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Tab
	}{
		{
			name:  "single tab without filter",
			input: "[All]",
			expected: []Tab{
				{Name: "All", Filter: ""},
			},
		},
		{
			name:  "three tabs with filters",
			input: "[All] | [Active] not done | [Done] done",
			expected: []Tab{
				{Name: "All", Filter: ""},
				{Name: "Active", Filter: "not done"},
				{Name: "Done", Filter: "done"},
			},
		},
		{
			name:  "tabs with comparison filters",
			input: "[High] priority = high | [Low] priority = low",
			expected: []Tab{
				{Name: "High", Filter: "priority = high"},
				{Name: "Low", Filter: "priority = low"},
			},
		},
		{
			name:     "no tabs - regular heading",
			input:    "Just a normal heading",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTabsFromContent(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tabs, got %d", len(tt.expected), len(result))
				return
			}
			for i, tab := range result {
				if tab.Name != tt.expected[i].Name {
					t.Errorf("tab %d: expected name %q, got %q", i, tt.expected[i].Name, tab.Name)
				}
				if tab.Filter != tt.expected[i].Filter {
					t.Errorf("tab %d: expected filter %q, got %q", i, tt.expected[i].Filter, tab.Filter)
				}
			}
		})
	}
}

func TestProcessTabbedHeadings(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectTabs    bool
		expectTabBar  bool
		expectFilters []string
	}{
		{
			name:          "heading with tabs",
			input:         `<h2 id="tasks">[All] | [Active] not done | [Done] done</h2>`,
			expectTabs:    true,
			expectTabBar:  true,
			expectFilters: []string{"", "not done", "done"},
		},
		{
			name:          "regular heading without tabs",
			input:         `<h2 id="intro">Introduction</h2>`,
			expectTabs:    false,
			expectTabBar:  false,
			expectFilters: nil,
		},
		{
			name:          "h3 with tabs",
			input:         `<h3 id="items">[Pending] | [Completed] completed</h3>`,
			expectTabs:    true,
			expectTabBar:  true,
			expectFilters: []string{"", "completed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processTabbedHeadings(tt.input)

			hasTabsClass := strings.Contains(result, "tinkerdown-tabs")
			hasTabBar := strings.Contains(result, "tinkerdown-tabs-bar")

			if hasTabsClass != tt.expectTabs {
				t.Errorf("expected tabs class: %v, got: %v\nResult: %s", tt.expectTabs, hasTabsClass, result)
			}
			if hasTabBar != tt.expectTabBar {
				t.Errorf("expected tab bar: %v, got: %v\nResult: %s", tt.expectTabBar, hasTabBar, result)
			}

			// Check filters are present
			for _, filter := range tt.expectFilters {
				if filter != "" {
					filterAttr := `data-filter="` + filter + `"`
					if !strings.Contains(result, filterAttr) {
						t.Errorf("expected filter attribute %q not found\nResult: %s", filterAttr, result)
					}
				}
			}
		})
	}
}
