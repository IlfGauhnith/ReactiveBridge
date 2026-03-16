package validator

// SchemaType is a custom type for our allowed list (Safe Enum pattern)
type SchemaType string

const (
	Fintech    SchemaType = "fintech"
	Healthcare SchemaType = "healthcare"
	Logistics  SchemaType = "logistics"
	Security   SchemaType = "security"
)

// IsValidSchema checks if the provided string matches one of our supported types.
// This is our first line of defense against garbage data or typos.
func IsValidSchema(t string) bool {
	switch SchemaType(t) {
	case Fintech, Healthcare, Logistics, Security:
		return true
	}
	return false
}
