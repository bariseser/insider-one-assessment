package utils

import (
	"encoding/json"
	"net/http"

	"insider-one-assessment/internal/constants"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type ErrorSchema struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HTTPErrorResponse struct {
	Timestamp int64       `json:"timestamp"`
	Error     ErrorSchema `json:"error"`
}

type HTTPSuccessResponse struct {
	Timestamp int64 `json:"timestamp"`
	Data      any   `json:"data"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteSuccess(w http.ResponseWriter, statusCode int, data any) {
	WriteJSON(w, statusCode, HTTPSuccessResponse{
		Timestamp: GetCurrentTimestamp(),
		Data:      data,
	})
}

func WriteError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	schema := ErrorSchema{
		Code:    constants.UnexpectedErrCode,
		Message: constants.UnexpectedMsg,
	}

	if errorBag, ok := err.(*ErrorBag); ok {
		statusCode = errorBag.GetStatusCode()
		schema.Code = errorBag.GetCode()
		schema.Message = errorBag.GetMessage()
	}

	WriteJSON(w, statusCode, HTTPErrorResponse{
		Timestamp: GetCurrentTimestamp(),
		Error:     schema,
	})
}
