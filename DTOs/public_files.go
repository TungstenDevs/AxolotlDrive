package dtos

type PaginationParams struct {
	Page  int `query:"page"`
	Limit int `query:"limit"`
}

type SearchParams struct {
	Q     string `query:"q"`
	Page  int    `query:"page"`
	Limit int    `query:"limit"`
}

type FileSystemItem struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	Size       int64   `json:"size"`
	IsDir      bool    `json:"is_dir"`
	CreatedAt  *int64  `json:"created_at,omitempty"`
	ModifiedAt *int64  `json:"modified_at,omitempty"`
	MimeType   *string `json:"mime_type,omitempty"`
	Etag       string  `json:"etag"`
}

type PaginatedItems struct {
	Items      []FileSystemItem `json:"items"`
	Total      int32            `json:"total"`
	Page       int32            `json:"page"`
	Limit      int32            `json:"limit"`
	TotalPages int32            `json:"total_pages"`
	HasNext    bool             `json:"has_next"`
	HasPrev    bool             `json:"has_prev"`
}

type ErrorResponse struct {
	Error     string  `json:"error"`
	Timestamp string  `json:"timestamp"`
	RequestID string  `json:"request_id"`
	Debug     *string `json:"debug,omitempty"`
}

type WebSocketMessage struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

type RenameRequest struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

type MoveRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type CopyRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}
