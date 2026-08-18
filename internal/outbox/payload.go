package outbox

import (
	"encoding/json"
	"fmt"
)

func MarshalPayload(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal outbox payload: %w", err)
	}

	return string(data), nil
}
