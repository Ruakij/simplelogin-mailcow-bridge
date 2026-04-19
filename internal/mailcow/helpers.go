package mailcow

import (
	"encoding/json"
	"io"
	"strings"
)

func readResponseBody(body io.Reader) ([]byte, error) {
	return io.ReadAll(body)
}

// stringList unmarshals a JSON value that may be either a JSON array of strings
// or a comma-separated string (as returned by some Mailcow API fields).
type stringList []string

func (s *stringList) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = nil
		return nil
	}

	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = nonEmptyTrimmed(arr)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = nonEmptyTrimmed(strings.Split(str, ","))
	return nil
}

func nonEmptyTrimmed(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
