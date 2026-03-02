package store

import "testing"

func TestMetricColumn(t *testing.T) {
	tests := []struct {
		metric  string
		wantCol string
		wantTab string
		wantErr bool
	}{
		{"recovery", "recovery_score", "recovery", false},
		{"hrv", "hrv_rmssd", "recovery", false},
		{"rhr", "resting_heart_rate", "recovery", false},
		{"strain", "strain", "cycle", false},
		{"sleep_duration", "(total_in_bed_time_milli - total_awake_time_milli)", "sleep", false},
		{"sleep_efficiency", "sleep_efficiency_pct", "sleep", false},
		{"invalid_metric", "", "", true},
	}

	for _, tt := range tests {
		col, tab, err := metricColumn(tt.metric)
		if tt.wantErr {
			if err == nil {
				t.Errorf("metricColumn(%q) expected error, got nil", tt.metric)
			}
			continue
		}
		if err != nil {
			t.Errorf("metricColumn(%q) unexpected error: %v", tt.metric, err)
			continue
		}
		if col != tt.wantCol || tab != tt.wantTab {
			t.Errorf("metricColumn(%q) = (%q, %q), want (%q, %q)",
				tt.metric, col, tab, tt.wantCol, tt.wantTab)
		}
	}
}

func TestDateColumn(t *testing.T) {
	tests := []struct {
		table string
		want  string
	}{
		{"recovery", "created_at"},
		{"cycle", "start"},
		{"sleep", "start"},
		{"unknown", "start"},
	}

	for _, tt := range tests {
		got := dateColumn(tt.table)
		if got != tt.want {
			t.Errorf("dateColumn(%q) = %q, want %q", tt.table, got, tt.want)
		}
	}
}
