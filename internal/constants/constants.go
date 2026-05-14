package constants

import "net/http"

const (
	NotFoundErrCode   = "not_found"
	ValidationErrCode = "validation_failed"
	UnexpectedErrCode = "unexpected_error"
	BodyParserErrCode = "body_parser_failed"
	ConflictErrCode   = "conflict"
	ProcessingErrCode = "processing_error"

	UnexpectedMsg = "An unexpected error has occurred."
	BodyParserMsg = "The given values could not be parsed."
)

var ErrorCodeHTTPStatusMap = map[string]int{
	NotFoundErrCode:   http.StatusNotFound,
	ValidationErrCode: http.StatusBadRequest,
	BodyParserErrCode: http.StatusBadRequest,
	ConflictErrCode:   http.StatusConflict,
	ProcessingErrCode: http.StatusInternalServerError,
	UnexpectedErrCode: http.StatusInternalServerError,
}

func HTTPStatusByErrorCode(code string) int {
	if statusCode, ok := ErrorCodeHTTPStatusMap[code]; ok {
		return statusCode
	}

	return http.StatusInternalServerError
}
