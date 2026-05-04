package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"git.infra.hisao.org/hisao/whooper/internal/config"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

const defaultAPILimit = 90
const maxAPILimit = 1000

func apiSummaryHandler(reporter statusReporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = writeStatusJSON(w, reporter())
	}
}

func apiRowsHandler(query string, args func(*http.Request) []any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		db, err := store.OpenReadOnly(config.DBPath())
		if err != nil {
			http.Error(w, fmt.Sprintf("open database: %v", err), http.StatusServiceUnavailable)
			return
		}
		defer db.Close()

		rows, err := queryJSONRows(db, query, args(r)...)
		if err != nil {
			http.Error(w, fmt.Sprintf("query database: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
	}
}

func queryJSONRows(db *store.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	out := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(cols))
		dest := make([]any, len(cols))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = normalizeSQLValue(values[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func normalizeSQLValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func apiLimit(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultAPILimit
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return defaultAPILimit
	}
	if limit > maxAPILimit {
		return maxAPILimit
	}
	return limit
}

func limitArg(r *http.Request) []any {
	return []any{apiLimit(r)}
}
