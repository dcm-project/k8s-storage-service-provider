package kubernetes

import "testing"

func TestNormalizePageSize(t *testing.T) {
	tests := []struct {
		in, want int32
	}{
		{0, defaultPageSize},
		{-1, defaultPageSize},
		{50, 50},
		{1000, 1000},
		{1001, maxPageSizeLimit},
		{5000, maxPageSizeLimit},
	}
	for _, tt := range tests {
		if got := normalizePageSize(tt.in); got != tt.want {
			t.Fatalf("normalizePageSize(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
