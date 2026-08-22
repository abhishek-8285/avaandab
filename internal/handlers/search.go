package handlers

import (
	"database/sql"
	"net/http"
	"strings"
)

// Global search across the core entities. Each section is gated by its own
// read permission so the result set never leaks resources the user cannot
// list. LIKE wildcards in user input are escaped (ESCAPE '\').

type SearchRow struct {
	ID    string
	Title string
	Sub   string
	Href  string
}

type SearchSection struct {
	Key   string
	Label string
	Rows  []SearchRow
	Total int
}

func likeEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func (a *App) canRead(r *http.Request, userID, resource string) bool {
	if a.AuthSrv == nil || userID == "" {
		return false
	}
	return a.AuthSrv.Can(userID, resource, "read")
}

// SearchPage renders GET /search?q= — grouped cross-entity results.
func (a *App) SearchPage(w http.ResponseWriter, r *http.Request) {
	session, _ := a.getUserFromContext(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	sections := []SearchSection{}
	if q != "" && a.DB != nil {
		like := "%" + likeEscape(q) + "%"
		userID := ""
		if session != nil {
			userID = session.UserID
		}

		type searchSpec struct {
			key, label, countSQL, rowSQL string
			args                         []any
			scan                         func(scan func(...any) error) (SearchRow, error)
		}

		specs := []searchSpec{
			{
				key: "bookings", label: "Bookings",
				countSQL: `SELECT COUNT(*) FROM bookings b LEFT JOIN customers c ON c.id = b.customer_id
					WHERE b.booking_number LIKE ? ESCAPE '\' OR c.name LIKE ? ESCAPE '\'`,
				rowSQL: `SELECT b.id, b.booking_number, b.status, COALESCE(c.name, '') FROM bookings b
					LEFT JOIN customers c ON c.id = b.customer_id
					WHERE b.booking_number LIKE ? ESCAPE '\' OR c.name LIKE ? ESCAPE '\'
					ORDER BY b.created_at DESC LIMIT 5`,
				args: []any{like, like},
				scan: func(scan func(...any) error) (SearchRow, error) {
					var row SearchRow
					var status, customer string
					if err := scan(&row.ID, &row.Title, &status, &customer); err != nil {
						return row, err
					}
					row.Sub = strings.Join(nonEmpty(status, customer), " · ")
					row.Href = "/bookings/" + row.ID
					return row, nil
				},
			},
			{
				key: "trips", label: "Trips",
				countSQL: `SELECT COUNT(*) FROM trips WHERE trip_number LIKE ? ESCAPE '\'`,
				rowSQL: `SELECT id, trip_number, status FROM trips
					WHERE trip_number LIKE ? ESCAPE '\' ORDER BY created_at DESC LIMIT 5`,
				args: []any{like},
				scan: func(scan func(...any) error) (SearchRow, error) {
					var row SearchRow
					var status string
					if err := scan(&row.ID, &row.Title, &status); err != nil {
						return row, err
					}
					row.Sub = status
					row.Href = "/trips/" + row.ID
					return row, nil
				},
			},
			{
				key: "vehicles", label: "Vehicles",
				countSQL: `SELECT COUNT(*) FROM vehicles WHERE registration_number LIKE ? ESCAPE '\' OR vehicle_number LIKE ? ESCAPE '\'`,
				rowSQL: `SELECT id, registration_number, COALESCE(vehicle_number, '') FROM vehicles
					WHERE registration_number LIKE ? ESCAPE '\' OR vehicle_number LIKE ? ESCAPE '\'
					ORDER BY registration_number LIMIT 5`,
				args: []any{like, like},
				scan: func(scan func(...any) error) (SearchRow, error) {
					var row SearchRow
					var num string
					if err := scan(&row.ID, &row.Title, &num); err != nil {
						return row, err
					}
					if num != "" && num != row.Title {
						row.Sub = num
					}
					row.Href = "/vehicles/" + row.ID
					return row, nil
				},
			},
			{
				key: "drivers", label: "Drivers",
				countSQL: `SELECT COUNT(*) FROM drivers WHERE first_name LIKE ? ESCAPE '\' OR last_name LIKE ? ESCAPE '\'
					OR phone LIKE ? ESCAPE '\' OR license_number LIKE ? ESCAPE '\'`,
				rowSQL: `SELECT id, first_name || ' ' || last_name, phone, license_number FROM drivers
					WHERE first_name LIKE ? ESCAPE '\' OR last_name LIKE ? ESCAPE '\' OR phone LIKE ? ESCAPE '\' OR license_number LIKE ? ESCAPE '\'
					ORDER BY first_name LIMIT 5`,
				args: []any{like, like, like, like},
				scan: func(scan func(...any) error) (SearchRow, error) {
					var row SearchRow
					var phone, licence string
					if err := scan(&row.ID, &row.Title, &phone, &licence); err != nil {
						return row, err
					}
					row.Sub = strings.Join(nonEmpty(phone, licence), " · ")
					row.Href = "/drivers/" + row.ID
					return row, nil
				},
			},
			{
				key: "customers", label: "Customers",
				countSQL: `SELECT COUNT(*) FROM customers WHERE name LIKE ? ESCAPE '\' OR COALESCE(company,'') LIKE ? ESCAPE '\'
					OR COALESCE(gst,'') LIKE ? ESCAPE '\' OR phone LIKE ? ESCAPE '\'`,
				rowSQL: `SELECT id, name, COALESCE(company, ''), COALESCE(gst, ''), phone FROM customers
					WHERE name LIKE ? ESCAPE '\' OR COALESCE(company,'') LIKE ? ESCAPE '\' OR COALESCE(gst,'') LIKE ? ESCAPE '\' OR phone LIKE ? ESCAPE '\'
					ORDER BY name LIMIT 5`,
				args: []any{like, like, like, like},
				scan: func(scan func(...any) error) (SearchRow, error) {
					var row SearchRow
					var company, gst, phone string
					if err := scan(&row.ID, &row.Title, &company, &gst, &phone); err != nil {
						return row, err
					}
					row.Sub = strings.Join(nonEmpty(company, gst, phone), " · ")
					row.Href = "/customers/" + row.ID
					return row, nil
				},
			},
			{
				key: "invoices", label: "Invoices",
				countSQL: `SELECT COUNT(*) FROM invoices WHERE invoice_number LIKE ? ESCAPE '\'`,
				rowSQL: `SELECT id, invoice_number, status FROM invoices
					WHERE invoice_number LIKE ? ESCAPE '\' ORDER BY created_at DESC LIMIT 5`,
				args: []any{like},
				scan: func(scan func(...any) error) (SearchRow, error) {
					var row SearchRow
					var status string
					if err := scan(&row.ID, &row.Title, &status); err != nil {
						return row, err
					}
					row.Sub = status
					row.Href = "/invoices/" + row.ID
					return row, nil
				},
			},
		}

		for _, spec := range specs {
			if !a.canRead(r, userID, spec.key) {
				continue
			}
			section := SearchSection{Key: spec.key, Label: spec.label}
			if err := a.DB.QueryRowContext(r.Context(), spec.countSQL, spec.args...).Scan(&section.Total); err != nil {
				section.Total = 0
				continue
			}
			if section.Total == 0 {
				continue
			}
			section.Rows = fetchRows(r, a.DB, spec.rowSQL, spec.args, spec.scan)
			sections = append(sections, section)
		}
	}

	a.renderPage(w, r, "search_results.html", PageData{
		Title: "Search",
		User:  session,
		Extra: map[string]interface{}{
			"Query":    q,
			"Sections": sections,
		},
	})
}

// fetchRows runs one search query and closes its rows via defer — sqlclosecheck
// requires defer even though errors here are non-fatal (section is skipped).
func fetchRows(r *http.Request, db *sql.DB, query string, args []any, scan func(func(...any) error) (SearchRow, error)) []SearchRow {
	rows, err := db.QueryContext(r.Context(), query, args...)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []SearchRow
	for rows.Next() {
		row, err := scan(rows.Scan)
		if err != nil {
			break
		}
		out = append(out, row)
	}
	return out
}

func nonEmpty(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
