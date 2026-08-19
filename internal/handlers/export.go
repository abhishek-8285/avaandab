package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
)

// writeCSV writes UTF-8 CSV with BOM and attachment headers.
func writeCSV(w http.ResponseWriter, filename string, header []string, rows [][]string, maxRows int, nextURL string) {
	if maxRows <= 0 {
		maxRows = 50000
	}

	truncated := false
	if len(rows) > maxRows {
		rows = rows[:maxRows]
		truncated = true
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("X-Export-Rows", strconv.Itoa(len(rows)))

	if truncated && nextURL != "" {
		w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL))
	}

	// UTF-8 BOM for Excel compatibility
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write(header)
	_ = cw.WriteAll(rows)
	cw.Flush()
}
