package custom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCustomToolArguments(t *testing.T) {
	tests := []struct {
		name     string
		innerXML string
		toolName string
		want     map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "empty xml",
			innerXML: "",
			toolName: "test-tool",
			want:     map[string]interface{}{},
			wantErr:  false,
		},
		{
			name: "single string parameter",
			innerXML: `<tool_name>recent-commits</tool_name>
			<format>oneline</format>`,
			toolName: "recent-commits",
			want: map[string]interface{}{
				"format": "oneline",
			},
			wantErr: false,
		},
		{
			name: "single integer parameter",
			innerXML: `<tool_name>recent-commits</tool_name>
			<count>20</count>`,
			toolName: "recent-commits",
			want: map[string]interface{}{
				"count": int64(20),
			},
			wantErr: false,
		},
		{
			name: "multiple parameters with different types",
			innerXML: `<tool_name>recent-commits</tool_name>
			<count>15</count>
			<format>full</format>
			<timeout>60</timeout>`,
			toolName: "recent-commits",
			want: map[string]interface{}{
				"count":  int64(15),
				"format": "full",
			},
			wantErr: false,
		},
		{
			name: "boolean parameter",
			innerXML: `<tool_name>test-tool</tool_name>
			<verbose>true</verbose>`,
			toolName: "test-tool",
			want: map[string]interface{}{
				"verbose": true,
			},
			wantErr: false,
		},
		{
			name: "float parameter",
			innerXML: `<tool_name>test-tool</tool_name>
			<threshold>3.14</threshold>`,
			toolName: "test-tool",
			want: map[string]interface{}{
				"threshold": 3.14,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCustomToolArguments([]byte(tt.innerXML))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
