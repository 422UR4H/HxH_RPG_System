package api

import (
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type ErrorModel struct {
	huma.ErrorModel
}

// func (em *ErrorModel) ContentType(ct string) string {
// 	return ct
// }

type TypeErr string

func (e TypeErr) Error() string {
	return string(e)
}

var NewErrorWithType = func(
	status int, detail string, errs ...error,
) huma.StatusError {

	var errorDetails []*huma.ErrorDetail
	var (
		t       string
		typeErr TypeErr
	)
	for _, e := range errs {
		if e == nil {
			continue
		}
		if casted, ok := e.(huma.ErrorDetailer); ok {
			errorDetails = append(errorDetails, casted.ErrorDetail())
		} else if errors.As(e, &typeErr) {
			t = typeErr.Error()
		} else {
			errorDetails = append(errorDetails, &huma.ErrorDetail{Message: e.Error()})
		}
	}
	return &ErrorModel{
		huma.ErrorModel{
			Type:   t,
			Title:  http.StatusText(status),
			Status: status,
			Detail: detail,
			Errors: errorDetails,
		},
	}
}
