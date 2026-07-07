package response

import (
	"encoding/json"
	"net/http"
)

// JSON записывает информацию в тело ответа
// JSON ResponseWriter - объект, через который handler пишет ответ
// JSON WriteHeader - записывает код ответа (OK, ERROR ...)
// JSON json.NewEncoder(w).Encode(data) - записывает ответ в формате json в response body
func JSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return
	}
}
