package ports

// IDGenerator provides a mockable interface for generating random IDs.
type IDGenerator interface {
	GenerateUUID() string
	GenerateDisplayID(prefix string) string
}
