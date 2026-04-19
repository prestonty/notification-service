package template

import "testing"

func TestRender(t *testing.T) {
	e := New()

	tests := []struct {
		name     string
		tmpl     string
		data     map[string]string
		expected string
		wantErr  bool
	}{
		{
			name:     "simple replacement",
			tmpl:     "Hello {name}",
			data:     map[string]string{"name": "Alice"},
			expected: "Hello Alice",
		},
		{
			name:     "multiple placeholders",
			tmpl:     "PR #{pr_id} was merged in {repo} by {author}.",
			data:     map[string]string{"pr_id": "42", "repo": "my-repo", "author": "alice"},
			expected: "PR #42 was merged in my-repo by alice.",
		},
		{
			name:     "no placeholders",
			tmpl:     "No placeholders here",
			data:     map[string]string{},
			expected: "No placeholders here",
		},
		{
			name:    "missing placeholder returns error",
			tmpl:    "Hello {name}",
			data:    map[string]string{},
			wantErr: true,
		},
		{
			name:    "partial data returns error",
			tmpl:    "Hello {name}, welcome to {place}",
			data:    map[string]string{"name": "Alice"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Render(tt.tmpl, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
