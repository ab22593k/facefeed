package cmd

// GraphAPIVersion is the Facebook Graph API version used for all API requests.
// Bump this to upgrade the API version across all endpoints.
const graphAPIVersion = "v25.0"

// ImageInput represents a single image to upload.
type ImageInput struct {
	Path     string
	Type     string // "url" or "file"
	Size     int64
	Filename string
}

// PublishTarget represents a Facebook page or group to publish to.
type PublishTarget struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// TargetResult holds the publish result for a single target.
type TargetResult struct {
	TargetID string
	Results  []UploadResult // for images
	PostID   string         // for text/link
	Error    error
}

// UploadResult holds the result of a single image upload.
type UploadResult struct {
	Filename string
	Success  bool
	PostID   string
	Error    error
}

// ValidationResult holds the outcome of input validation.
type ValidationResult struct {
	Valid     bool
	Images    []ImageInput
	Errors    []string
	Warnings  []string
	TotalSize int64
}
