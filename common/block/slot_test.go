package block

import "testing"

func TestParseSlotID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       SlotID
		wantErr    bool
		wantString string
	}{
		{
			name:  "controller enclosure slot",
			input: "0:24:15",
			want: SlotID{
				ControllerID: "0",
				EnclosureID:  "24",
				SlotID:       "15",
			},
			wantString: "0:24:15",
		},
		{
			name:  "controller slot",
			input: "1:7",
			want: SlotID{
				ControllerID: "1",
				SlotID:       "7",
			},
			wantString: "1:7",
		},
		{name: "empty", input: "", wantErr: true},
		{name: "missing part", input: "0::1", wantErr: true},
		{name: "too many parts", input: "0:1:2:3", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSlotID(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSlotID(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSlotID(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ParseSlotID(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
			if got.String() != tt.wantString {
				t.Fatalf("slot.String() = %q, want %q", got.String(), tt.wantString)
			}
		})
	}
}
