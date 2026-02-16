package local

import (
	"testing"
)

func TestValidateRepositoryName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		// Valid names
		{
			name:    "simple lowercase name",
			input:   "myrepo",
			wantErr: false,
		},
		{
			name:    "name with hyphens",
			input:   "my-repo-name",
			wantErr: false,
		},
		{
			name:    "name with numbers",
			input:   "repo123",
			wantErr: false,
		},
		{
			name:    "single character",
			input:   "a",
			wantErr: false,
		},
		{
			name:    "numbers only",
			input:   "123",
			wantErr: false,
		},
		{
			name:    "alphanumeric mix with hyphens",
			input:   "hello-world-42",
			wantErr: false,
		},
		{
			name:    "two characters",
			input:   "ab",
			wantErr: false,
		},
		{
			name:    "number and letter",
			input:   "1a",
			wantErr: false,
		},

		// Invalid: contains slash
		{
			name:    "contains slash",
			input:   "my/repo",
			wantErr: true,
			errMsg:  "cannot contain slashes",
		},

		// Invalid: contains underscore
		{
			name:    "contains underscore",
			input:   "my_repo",
			wantErr: true,
			errMsg:  "cannot contain underscores",
		},
		{
			name:    "underscore at start",
			input:   "_repo",
			wantErr: true,
			errMsg:  "cannot contain underscores",
		},
		{
			name:    "underscore at end",
			input:   "repo_",
			wantErr: true,
			errMsg:  "cannot contain underscores",
		},

		// Invalid: uppercase letters
		{
			name:    "uppercase letters",
			input:   "MyRepo",
			wantErr: true,
			errMsg:  "must be lowercase",
		},
		{
			name:    "all uppercase",
			input:   "MYREPO",
			wantErr: true,
			errMsg:  "must be lowercase",
		},
		{
			name:    "mixed case with hyphen",
			input:   "My-Repo",
			wantErr: true,
			errMsg:  "must be lowercase",
		},

		// Invalid: starts or ends with hyphen
		{
			name:    "starts with hyphen",
			input:   "-repo",
			wantErr: true,
			errMsg:  "cannot start or end with hyphen",
		},
		{
			name:    "ends with hyphen",
			input:   "repo-",
			wantErr: true,
			errMsg:  "cannot start or end with hyphen",
		},
		{
			name:    "only a hyphen",
			input:   "-",
			wantErr: true,
			errMsg:  "cannot start or end with hyphen",
		},

		// Invalid: empty string
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "must contain only lowercase letters",
		},

		// Invalid: special characters
		{
			name:    "contains space",
			input:   "my repo",
			wantErr: true,
			errMsg:  "must contain only lowercase letters",
		},
		{
			name:    "contains dot",
			input:   "my.repo",
			wantErr: true,
			errMsg:  "must contain only lowercase letters",
		},
		{
			name:    "contains at sign",
			input:   "my@repo",
			wantErr: true,
			errMsg:  "must contain only lowercase letters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepositoryName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRepositoryName(%q) = nil, want error containing %q", tt.input, tt.errMsg)
					return
				}
				if tt.errMsg != "" {
					errStr := err.Error()
					if !containsSubstring(errStr, tt.errMsg) {
						t.Errorf("validateRepositoryName(%q) error = %q, want error containing %q", tt.input, errStr, tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("validateRepositoryName(%q) = %v, want nil", tt.input, err)
				}
			}
		})
	}
}

// containsSubstring checks if s contains substr (avoids importing strings in test)
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
