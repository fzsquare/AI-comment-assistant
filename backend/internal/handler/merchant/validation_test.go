package merchant

import "testing"

func TestValidateHTTPURLRejectsUnsafeSchemes(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "data:text/plain,hi", "file:///etc/passwd", "//example.com/path"} {
		if err := validateHTTPURL(raw, true); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestValidateHTTPURLAllowsHTTPAndHTTPS(t *testing.T) {
	for _, raw := range []string{"http://example.com/image.jpg", "https://example.com/path?q=1"} {
		if err := validateHTTPURL(raw, true); err != nil {
			t.Fatalf("expected %q to be allowed, got %v", raw, err)
		}
	}
}

func TestNormalizeTargetCountBounds(t *testing.T) {
	tests := []struct {
		name       string
		input      int
		defaultVal int
		maxVal     int
		want       int
		wantErr    bool
	}{
		{name: "uses default for zero", input: 0, defaultVal: 10, maxVal: 50, want: 10},
		{name: "allows positive within max", input: 25, defaultVal: 10, maxVal: 50, want: 25},
		{name: "rejects above max", input: 51, defaultVal: 10, maxVal: 50, wantErr: true},
		{name: "rejects negative", input: -1, defaultVal: 10, maxVal: 50, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTargetCount(tt.input, tt.defaultVal, tt.maxVal)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
