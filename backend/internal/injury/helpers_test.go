package injury

import "context"

// stubGen returns a Generate seam yielding canned bytes (or an error) and records
// the schema that was requested, so the LLM-backed parser and assist can be tested
// table-driven without touching Claude.
func stubGen(out string, err error, gotSchema *map[string]any) Generate {
	return func(_ context.Context, _ string, _ []byte, schema map[string]any) ([]byte, error) {
		if gotSchema != nil {
			*gotSchema = schema
		}
		if err != nil {
			return nil, err
		}
		return []byte(out), nil
	}
}

// equalStrings reports element-wise equality, treating nil and empty as equal.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
