package dto

// ErrorResponse — единый формат ошибок для всех эндпоинтов:
//
//	{"error": "validation failed", "details": {"email": "must be a valid email"}}
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}
