package cmd

import (
	"strings"
	"testing"
)

func TestSyncSinceValidation(t *testing.T) {
	tests := []struct {
		name    string
		since   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid date",
			since:   "2024-01-15",
			wantErr: false,
		},
		{
			name:    "invalid format",
			since:   "01-15-2024",
			wantErr: true,
			errMsg:  "must be YYYY-MM-DD",
		},
		{
			name:    "nonsense string",
			since:   "not-a-date",
			wantErr: true,
			errMsg:  "must be YYYY-MM-DD",
		},
		{
			name:    "partial date",
			since:   "2024-01",
			wantErr: true,
			errMsg:  "must be YYYY-MM-DD",
		},
		{
			name:    "invalid calendar date",
			since:   "2024-02-30",
			wantErr: true,
			errMsg:  "must be YYYY-MM-DD",
		},
		{
			name:    "empty date",
			since:   "",
			wantErr: false, // empty is valid, it means default sync
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSyncSince(tt.since)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateSyncSince() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSyncSince() error = %v, wantErrMsg %v", err, tt.errMsg)
			}
		})
	}
}
