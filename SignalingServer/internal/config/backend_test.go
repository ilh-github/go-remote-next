package config

import "testing"

func TestResolveRedisDriver(t *testing.T) {
	tests := []struct {
		name          string
		database      string
		configured    string
		wantRedisMode string
	}{
		{
			name:          "SQLite defaults to in-process Redis",
			database:      "SQLite",
			wantRedisMode: "embedded",
		},
		{
			name:          "PostgreSQL keeps external Redis",
			database:      "postgres",
			wantRedisMode: "redis",
		},
		{
			name:          "explicit Redis wins over SQLite default",
			database:      "sqlite",
			configured:    "redis",
			wantRedisMode: "redis",
		},
		{
			name:          "explicit embedded value is normalized",
			database:      "postgres",
			configured:    "  EMBEDDED ",
			wantRedisMode: "embedded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveRedisDriver(tt.database, tt.configured); got != tt.wantRedisMode {
				t.Fatalf("resolveRedisDriver(%q, %q) = %q, want %q",
					tt.database, tt.configured, got, tt.wantRedisMode)
			}
		})
	}
}
