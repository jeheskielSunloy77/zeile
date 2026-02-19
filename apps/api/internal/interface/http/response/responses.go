package response

type Response[T any] struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
	Success bool   `json:"success"`

	Data *T `json:"data,omitempty"`
}

type PaginatedResponse[T any] struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
	Success bool   `json:"success"`

	Data       []T `json:"data"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

func NewPaginatedResponse[T any](message string, entities []T, total int64, limit int, offset int) PaginatedResponse[T] {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	page := offset/limit + 1

	return PaginatedResponse[T]{
		Status:     200,
		Success:    true,
		Message:    message,
		Data:       entities,
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: totalPages,
	}
}
