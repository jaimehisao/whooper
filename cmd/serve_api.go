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

type errorResponse struct {
	Error string `json:"error"`
}

func writeErrorJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(errorResponse{Error: message})
}

func apiSummaryHandler(reporter statusReporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErrorJSON(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = writeStatusJSON(w, reporter())
	}
}

func apiRowsHandler(view, dateCol, orderCol string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErrorJSON(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		db, err := store.OpenReadOnly(config.DBPath())
		if err != nil {
			writeErrorJSON(w, fmt.Sprintf("open database: %v", err), http.StatusServiceUnavailable)
			return
		}
		defer db.Close()

		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if err := validateDateRange(from, to); err != nil {
			writeErrorJSON(w, err.Error(), http.StatusBadRequest)
			return
		}
		limit := apiLimit(r)

		query, args := apiRowsQuery(view, dateCol, orderCol, from, to, limit)
		rows, err := queryJSONRows(db, query, args...)
		if err != nil {
			writeErrorJSON(w, fmt.Sprintf("query database: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rows)
	}
}

func apiRowsQuery(view, dateCol, orderCol, from, to string, limit int) (string, []any) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE 1=1", view)
	var args []any

	if from != "" {
		query += fmt.Sprintf(" AND %s >= ?", dateCol)
		args = append(args, from)
	}
	if to != "" {
		query += fmt.Sprintf(" AND %s <= ?", dateCol)
		args = append(args, to)
	}

	query += fmt.Sprintf(" ORDER BY %s DESC LIMIT ?", orderCol)
	args = append(args, limit)
	return query, args
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
