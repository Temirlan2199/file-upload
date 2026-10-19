package dto

// ErrorResponse — единый формат ошибок API: {"error": "..."}
type ErrorResponse struct {
	Error string `json:"error"`
}
