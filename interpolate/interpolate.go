package interpolate

import (
	"regexp"
)

var pattern = regexp.MustCompile(`\{\{\s*(\w+)\s*\}\}`)

// Apply replaces all {{ VAR }} occurrences in s with values from vars.
// Both env variables and captured test variables share the same flat namespace.
func Apply(s string, vars map[string]string) string {
	return pattern.ReplaceAllStringFunc(s, func(match string) string {
		key := pattern.FindStringSubmatch(match)[1]

		if val, ok := vars[key]; ok {
			return val
		}

		return match // leave unreplaced if not found
	})
}

// ApplyToMap applies interpolation to all string values in a map.
func ApplyToMap(m map[string]string, vars map[string]string) map[string]string {
	result := make(map[string]string, len(m))

	for k, v := range m {
		result[k] = Apply(v, vars)
	}

	return result
}
