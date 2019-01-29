package httputil

import (
	"encoding/json"
	"net/http"
)

type HttpError struct {
	code    int    `json:"-"`
	errInfo string `json:"error"`
}

func (err *HttpError) Code() int {
	return err.code
}

func (err *HttpError) Error() string {
	return err.errInfo
}

func ReplyError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *HttpError:
		w.WriteHeader(e.Code())
		content, _ := json.Marshal(e)
		w.Write(content)
	default:
		w.WriteHeader(599)
		w.Write([]byte(err.Error()))
	}
}

func NewHttpError(code int, errInfo string) *HttpError {
	return &HttpError{
		code:    code,
		errInfo: errInfo,
	}
}
