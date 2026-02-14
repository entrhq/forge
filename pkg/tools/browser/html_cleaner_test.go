package browser

import (
	"strings"
	"testing"
)

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		wantTitle string
		wantDesc  string
		wantHTML  []string // substrings that should be present
		wantNot   []string // substrings that should NOT be present
		truncated bool
	}{
		{
			name: "basic HTML with script and style removal",
			input: `<html>
				<head>
					<title>Test Page</title>
					<meta name="description" content="Test description">
					<script>alert('evil');</script>
					<style>body { color: red; }</style>
				</head>
				<body>
					<h1 id="main-title">Hello World</h1>
					<p class="intro">This is a test.</p>
				</body>
			</html>`,
			maxLength: 10000,
			wantTitle: "Test Page",
			wantDesc:  "Test description",
			wantHTML:  []string{"<h1 id=\"main-title\">", "Hello World", "<p class=\"intro\">", "This is a test"},
			wantNot:   []string{"<script>", "alert", "<style>", "color: red"},
			truncated: false,
		},
		{
			name: "preserve semantic structure",
			input: `<html><body>
				<header><nav><a href="/home">Home</a></nav></header>
				<main>
					<section id="content">
						<article><h2>Article Title</h2></article>
					</section>
				</main>
				<footer><p>Footer</p></footer>
			</body></html>`,
			maxLength: 10000,
			wantHTML:  []string{"<header>", "<nav>", "<main>", "<section id=\"content\">", "<article>", "<footer>"},
			truncated: false,
		},
		{
			name: "preserve important attributes",
			input: `<html><body>
				<form action="/submit" method="post">
					<input type="text" name="username" id="user-input" placeholder="Enter name" data-test="username-field">
					<button type="submit" class="btn-primary">Submit</button>
				</form>
			</body></html>`,
			maxLength: 10000,
			wantHTML: []string{
				`<form action="/submit" method="post">`,
				`type="text"`,
				`name="username"`,
				`id="user-input"`,
				`placeholder="Enter name"`,
				`data-test="username-field"`,
				`class="btn-primary"`,
			},
			truncated: false,
		},
		{
			name: "remove unwanted elements",
			input: `<html><body>
				<div>Content</div>
				<script src="app.js"></script>
				<noscript>No JS</noscript>
				<iframe src="ad.html"></iframe>
				<svg><circle/></svg>
			</body></html>`,
			maxLength: 10000,
			wantHTML:  []string{"<div>", "Content"},
			wantNot:   []string{"<script>", "<noscript>", "<iframe>", "<svg>", "No JS"},
			truncated: false,
		},
		{
			name: "truncate at boundary",
			input: `<html><body>
				<p>First paragraph with some content.</p>
				<p>Second paragraph with more content.</p>
				<p>Third paragraph that should be truncated.</p>
			</body></html>`,
			maxLength: 100,
			wantHTML:  []string{"First paragraph"},
			truncated: true,
		},
		{
			name: "preserve links with href",
			input: `<html><body>
				<a href="https://example.com" target="_blank" class="external">Link Text</a>
			</body></html>`,
			maxLength: 10000,
			wantHTML:  []string{`href="https://example.com"`, `target="_blank"`, `class="external"`, "Link Text"},
			truncated: false,
		},
		{
			name: "preserve table structure",
			input: `<html><body>
				<table summary="Data table">
					<tr>
						<th>Header</th>
					</tr>
					<tr>
						<td>Cell</td>
					</tr>
				</table>
			</body></html>`,
			maxLength: 10000,
			wantHTML:  []string{`<table summary="Data table">`, "<tr>", "<th>", "<td>"},
			truncated: false,
		},
		{
			name: "handle void elements",
			input: `<html><body>
				<img src="test.jpg" alt="Test image">
				<br>
				<input type="text" name="field">
				<hr>
			</body></html>`,
			maxLength: 10000,
			wantHTML:  []string{`<img src="test.jpg" alt="Test image">`, "<br>", `<input type="text" name="field">`, "<hr>"},
			wantNot:   []string{"</img>", "</br>", "</input>", "</hr>"},
			truncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cleanHTML(tt.input, tt.maxLength)
			if err != nil {
				t.Fatalf("cleanHTML() error = %v", err)
			}

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}

			if result.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", result.Description, tt.wantDesc)
			}

			if result.Truncated != tt.truncated {
				t.Errorf("Truncated = %v, want %v", result.Truncated, tt.truncated)
			}

			for _, want := range tt.wantHTML {
				if !strings.Contains(result.HTML, want) {
					t.Errorf("HTML missing expected substring: %q\nGot: %s", want, result.HTML)
				}
			}

			for _, notWant := range tt.wantNot {
				if strings.Contains(result.HTML, notWant) {
					t.Errorf("HTML contains unwanted substring: %q\nGot: %s", notWant, result.HTML)
				}
			}
		})
	}
}

func TestIsSkippedElement(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"script", true},
		{"style", true},
		{"noscript", true},
		{"iframe", true},
		{"svg", true},
		{"div", false},
		{"p", false},
		{"span", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := isSkippedElement(tt.tag); got != tt.want {
				t.Errorf("isSkippedElement(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestIsBlockElement(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"div", true},
		{"p", true},
		{"section", true},
		{"h1", true},
		{"ul", true},
		{"table", true},
		{"span", false},
		{"a", false},
		{"strong", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			if got := isBlockElement(tt.tag); got != tt.want {
				t.Errorf("isBlockElement(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestShouldPreserveAttribute(t *testing.T) {
	tests := []struct {
		tag  string
		attr string
		want bool
	}{
		{"div", "id", true},
		{"div", "class", true},
		{"div", "style", false},
		{"div", "onclick", false},
		{"div", "data-test", true},
		{"a", "href", true},
		{"a", "target", true},
		{"img", "src", true},
		{"img", "alt", true},
		{"input", "name", true},
		{"input", "type", true},
		{"input", "placeholder", true},
		{"form", "action", true},
		{"form", "method", true},
	}

	for _, tt := range tests {
		t.Run(tt.tag+"_"+tt.attr, func(t *testing.T) {
			if got := shouldPreserveAttribute(tt.tag, tt.attr); got != tt.want {
				t.Errorf("shouldPreserveAttribute(%q, %q) = %v, want %v", tt.tag, tt.attr, got, tt.want)
			}
		})
	}
}
