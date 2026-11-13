package config

import "testing"

func TestRawPersistenceWorkerConfiguration(t *testing.T) {
	for _, tc := range []struct {
		value   string
		want    int
		invalid bool
	}{
		{"", 4, false}, {"1", 1, false}, {"8", 8, false},
		{"0", 0, true}, {"-1", 0, true}, {"four", 0, true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("RAW_PERSISTENCE_WORKERS", tc.value)
			cfg, err := Load()
			got := cfg.RawPersistenceWorkers
			if got != tc.want || (err != nil) != tc.invalid {
				t.Fatalf("workers=%d err=%v, want %d invalid=%v", got, err, tc.want, tc.invalid)
			}
		})
	}
}
