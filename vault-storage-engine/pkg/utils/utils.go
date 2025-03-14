package utils

import "fmt"

// Helper function to convert a slice to a map
func ConvertSliceToMap(slice []string) map[string]string {
	result := make(map[string]string)
	for i, v := range slice {
		key := fmt.Sprintf("key_%d", i) // Use a suitable key generation logic
		result[key] = v
	}
	return result
}
