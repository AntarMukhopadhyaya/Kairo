package frontend

// Diagnostic represents a user-facing parse/compile issue.
type Diagnostic struct {
	Message string
	Phase   string
	Line    int
	Column  int
}
