package domain

import "testing"

func TestValidateCPF(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{name: "plain digits", input: "52998224725", want: "52998224725"},
		{name: "formatted cpf", input: "529.982.247-25", want: "52998224725"},
		{name: "invalid check digit", input: "52998224724", wantError: true},
		{name: "all repeated digits", input: "111.111.111-11", wantError: true},
		{name: "wrong length", input: "123", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateCPF(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateCPF returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
