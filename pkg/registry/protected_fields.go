package registry

import "strings"

// protectedFields is a set of core field names that should not be stored in metadata
var protectedFields = map[string]bool{
	"id":        true,
	"name":      true,
	"active":    true,
	"createdAt": true,
	"updatedAt": true,
}

// isProtectedField checks if a metadata key is a protected core field
func isProtectedField(key string) bool {
	// Check for exact match first
	if protectedFields[key] {
		return true
	}
	// Check for case-insensitive match
	lowerKey := strings.ToLower(key)
	for k := range protectedFields {
		if strings.ToLower(k) == lowerKey {
			return true
		}
	}
	return false
}
