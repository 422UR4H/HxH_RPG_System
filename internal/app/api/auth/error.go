package auth

import (
	"fmt"
	"io"
)

// WriteAuthError writes a standardized authentication error response in JSON format
func WriteAuthError(w io.Writer, errorType string, message string) {
	w.Write(fmt.Appendf(nil, `{"error": "%s", "message": "%s"}`, errorType, message))
}
