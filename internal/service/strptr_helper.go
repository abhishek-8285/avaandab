package service

// strPtr returns a pointer to the given string, or nil if empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StrPtr returns a pointer to the given string, or nil if empty.
func StrPtr(s string) *string {
	return strPtr(s)
}
