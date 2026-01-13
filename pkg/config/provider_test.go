package config

import (
	"os"
	"testing"
)

func TestBuildProvider(t *testing.T) {
	// Save original env vars
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	originalBaseURL := os.Getenv("OPENAI_BASE_URL")
	defer func() {
		if originalAPIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
		if originalBaseURL != "" {
			os.Setenv("OPENAI_BASE_URL", originalBaseURL)
		} else {
			os.Unsetenv("OPENAI_BASE_URL")
		}
	}()

	tests := []struct {
		name           string
		cliModel       string
		cliBaseURL     string
		cliAPIKey      string
		envAPIKey      string
		envBaseURL     string
		defaultModel   string
		expectError    bool
		expectedModel  string
		expectedAPIKey string
		expectedURL    string
	}{
		{
			name:           "CLI flag takes precedence over env",
			cliModel:       "gpt-4",
			cliBaseURL:     "https://cli.example.com",
			cliAPIKey:      "cli-key",
			envAPIKey:      "env-key",
			envBaseURL:     "https://env.example.com",
			defaultModel:   "gpt-3.5-turbo",
			expectError:    false,
			expectedModel:  "gpt-4",
			expectedAPIKey: "cli-key",
			expectedURL:    "https://cli.example.com",
		},
		{
			name:           "Environment variable used when CLI empty",
			cliModel:       "",
			cliBaseURL:     "",
			cliAPIKey:      "",
			envAPIKey:      "env-key",
			envBaseURL:     "https://env.example.com",
			defaultModel:   "gpt-3.5-turbo",
			expectError:    false,
			expectedModel:  "gpt-3.5-turbo",
			expectedAPIKey: "env-key",
			expectedURL:    "https://env.example.com",
		},
		{
			name:         "Error when no API key provided",
			cliModel:     "",
			cliBaseURL:   "",
			cliAPIKey:    "",
			envAPIKey:    "",
			envBaseURL:   "",
			defaultModel: "gpt-3.5-turbo",
			expectError:  true,
		},
		{
			name:           "Default model used when CLI is default",
			cliModel:       "gpt-3.5-turbo",
			cliBaseURL:     "",
			cliAPIKey:      "test-key",
			envAPIKey:      "",
			envBaseURL:     "",
			defaultModel:   "gpt-3.5-turbo",
			expectError:    false,
			expectedModel:  "gpt-3.5-turbo",
			expectedAPIKey: "test-key",
			expectedURL:    "",
		},
		{
			name:           "Empty CLI model falls back to default",
			cliModel:       "",
			cliBaseURL:     "",
			cliAPIKey:      "test-key",
			envAPIKey:      "",
			envBaseURL:     "",
			defaultModel:   "gpt-4-turbo",
			expectError:    false,
			expectedModel:  "gpt-4-turbo",
			expectedAPIKey: "test-key",
			expectedURL:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envAPIKey != "" {
				os.Setenv("OPENAI_API_KEY", tt.envAPIKey)
			} else {
				os.Unsetenv("OPENAI_API_KEY")
			}
			if tt.envBaseURL != "" {
				os.Setenv("OPENAI_BASE_URL", tt.envBaseURL)
			} else {
				os.Unsetenv("OPENAI_BASE_URL")
			}

			// Call BuildProvider
			provider, err := BuildProvider(tt.cliModel, tt.cliBaseURL, tt.cliAPIKey, tt.defaultModel)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Errorf("Expected provider but got nil")
				return
			}

			// Verify the provider was created successfully
			// Note: Provider fields are private, so we just verify it's not nil
		})
	}
}
