package utils

import (
	"testing"
)

func TestCleanLLMJSON(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal JSON",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with plain markdown block",
			input:    "```\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with spaces and newlines",
			input:    "  \n```json\n{\"key\": \"value\"}\n```\n  ",
			expected: `{"key": "value"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanLLMJSON(tc.input)
			if result != tc.expected {
				t.Errorf("CleanLLMJSON() = %q; want %q", result, tc.expected)
			}
		})
	}
}

func TestCleanURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal URL", "https://example.com", "https://example.com"},
		{"URL with backticks", "`https://example.com`", "https://example.com"},
		{"URL with spaces", " https://example.com ", "https://example.com"},
		{"URL with mixed characters", " ` https://example.com ` ", "https://example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanURL(tc.input)
			if result != tc.expected {
				t.Errorf("CleanURL() = %q; want %q", result, tc.expected)
			}
		})
	}
}
