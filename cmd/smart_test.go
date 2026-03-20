package cmd

import "testing"

func TestSmartCommandHasUsableOutput(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		output   string
		want     bool
	}{
		{
			name:     "smartctl warning bit with stdout is accepted",
			exitCode: 4,
			output:   "SMART overall-health self-assessment test result: PASSED",
			want:     true,
		},
		{
			name:     "device open failure stays fatal",
			exitCode: 2,
			output:   "failed to open device",
			want:     false,
		},
		{
			name:     "empty output stays fatal",
			exitCode: 4,
			output:   "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smartCommandHasUsableOutput(tt.exitCode, tt.output)
			if got != tt.want {
				t.Fatalf("smartCommandHasUsableOutput(%d, %q) = %v, want %v", tt.exitCode, tt.output, got, tt.want)
			}
		})
	}
}
