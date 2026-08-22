package apperr

import "net/http"

const (
	CodeInternal               = "INTERNAL_ERROR"
	CodeNotImplemented         = "NOT_IMPLEMENTED"
	CodeServiceUnavailable     = "SERVICE_UNAVAILABLE"
	CodeValidation             = "VALIDATION_ERROR"
	CodeMalformedJSON          = "MALFORMED_JSON"
	CodeMissingField           = "MISSING_FIELD"
	CodeUnauthorized           = "UNAUTHORIZED"
	CodeTokenInvalid           = "TOKEN_INVALID"
	CodeTokenExpired           = "TOKEN_EXPIRED"
	CodeCSRF                   = "CSRF_FAILURE"
	CodeForbidden              = "FORBIDDEN"
	CodeNotFound               = "NOT_FOUND"
	CodeMethodNotAllowed       = "METHOD_NOT_ALLOWED"
	CodeConflict               = "CONFLICT"
	CodeRateLimited            = "RATE_LIMIT_EXCEEDED"
	CodePayloadTooLarge        = "PAYLOAD_TOO_LARGE"
	CodeUnsupportedMediaType   = "UNSUPPORTED_MEDIA_TYPE"
	CodeBookingNotFound        = "BOOKING_NOT_FOUND"
	CodeBookingInvalidState    = "BOOKING_INVALID_STATE"
	CodeDriverUnavailable      = "DRIVER_UNAVAILABLE"
	CodeVehicleUnavailable     = "VEHICLE_UNAVAILABLE"
	CodePaymentAlreadyRecorded = "PAYMENT_ALREADY_RECORDED"
	CodeKharchaApprovalDenied  = "KHARCHA_APPROVAL_DENIED"
	CodeKharchaAlreadyApproved = "KHARCHA_ALREADY_APPROVED"
	CodeFileTooLarge           = "FILE_TOO_LARGE"
)

var registry = map[string]*AppError{
	CodeInternal: {
		Code:       CodeInternal,
		HTTPStatus: http.StatusInternalServerError,
		Title:      "Internal Server Error",
		UserMsg:    "Something went wrong on our side. Please try again, or contact support if it keeps happening.",
	},
	CodeNotImplemented: {
		Code:       CodeNotImplemented,
		HTTPStatus: http.StatusNotImplemented,
		Title:      "Not Implemented",
		UserMsg:    "This feature is not available yet.",
	},
	CodeServiceUnavailable: {
		Code:       CodeServiceUnavailable,
		HTTPStatus: http.StatusServiceUnavailable,
		Title:      "Service Unavailable",
		UserMsg:    "We're briefly offline for maintenance. Please try again in a few minutes.",
	},
	CodeValidation: {
		Code:       CodeValidation,
		HTTPStatus: http.StatusBadRequest,
		Title:      "Validation Error",
		UserMsg:    "Please check the highlighted fields and try again.",
	},
	CodeMalformedJSON: {
		Code:       CodeMalformedJSON,
		HTTPStatus: http.StatusBadRequest,
		Title:      "Invalid Request Body",
		UserMsg:    "The request could not be read. Please refresh the page and try again.",
	},
	CodeMissingField: {
		Code:       CodeMissingField,
		HTTPStatus: http.StatusBadRequest,
		Title:      "Missing Information",
		UserMsg:    "Some required information is missing. Please fill in all required fields.",
	},
	CodeUnauthorized: {
		Code:       CodeUnauthorized,
		HTTPStatus: http.StatusUnauthorized,
		Title:      "Not Signed In",
		UserMsg:    "Please sign in to continue.",
	},
	CodeTokenInvalid: {
		Code:       CodeTokenInvalid,
		HTTPStatus: http.StatusUnauthorized,
		Title:      "Invalid Credentials",
		UserMsg:    "Your session is no longer valid. Please sign in again.",
	},
	CodeTokenExpired: {
		Code:       CodeTokenExpired,
		HTTPStatus: http.StatusUnauthorized,
		Title:      "Session Expired",
		UserMsg:    "Your session has expired. Please sign in again.",
	},
	CodeCSRF: {
		Code:       CodeCSRF,
		HTTPStatus: http.StatusForbidden,
		Title:      "Request Blocked",
		UserMsg:    "For your security this request was blocked. Please refresh the page and try again.",
	},
	CodeForbidden: {
		Code:       CodeForbidden,
		HTTPStatus: http.StatusForbidden,
		Title:      "Permission Denied",
		UserMsg:    "You don't have access to perform this action. Ask an admin if you need it.",
	},
	CodeNotFound: {
		Code:       CodeNotFound,
		HTTPStatus: http.StatusNotFound,
		Title:      "Not Found",
		UserMsg:    "We couldn't find what you were looking for. It may have been deleted or the link is wrong.",
	},
	CodeMethodNotAllowed: {
		Code:       CodeMethodNotAllowed,
		HTTPStatus: http.StatusMethodNotAllowed,
		Title:      "Method Not Allowed",
		UserMsg:    "That request isn't supported here. Please go back and try again.",
	},
	CodeConflict: {
		Code:       CodeConflict,
		HTTPStatus: http.StatusConflict,
		Title:      "Conflict",
		UserMsg:    "This conflicts with a recent change. Please refresh and try again.",
	},
	CodeRateLimited: {
		Code:       CodeRateLimited,
		HTTPStatus: http.StatusTooManyRequests,
		Title:      "Too Many Requests",
		UserMsg:    "You're doing that too fast. Please wait a moment and try again.",
	},
	CodePayloadTooLarge: {
		Code:       CodePayloadTooLarge,
		HTTPStatus: http.StatusRequestEntityTooLarge,
		Title:      "Request Too Large",
		UserMsg:    "That upload is too large. Please try a smaller file (max 32MB).",
	},
	CodeUnsupportedMediaType: {
		Code:       CodeUnsupportedMediaType,
		HTTPStatus: http.StatusUnsupportedMediaType,
		Title:      "Unsupported File Type",
		UserMsg:    "That file type isn't supported. Please try a different format.",
	},
	CodeBookingNotFound: {
		Code:       CodeBookingNotFound,
		HTTPStatus: http.StatusNotFound,
		Title:      "Booking Not Found",
		UserMsg:    "We couldn't find that booking. It may have been removed.",
	},
	CodeBookingInvalidState: {
		Code:       CodeBookingInvalidState,
		HTTPStatus: http.StatusConflict,
		Title:      "Booking Cannot Be Changed",
		UserMsg:    "This booking can no longer be changed in that way because of its current status.",
	},
	CodeDriverUnavailable: {
		Code:       CodeDriverUnavailable,
		HTTPStatus: http.StatusConflict,
		Title:      "Driver Unavailable",
		UserMsg:    "That driver isn't available for this trip. Please pick another driver.",
	},
	CodeVehicleUnavailable: {
		Code:       CodeVehicleUnavailable,
		HTTPStatus: http.StatusConflict,
		Title:      "Vehicle Unavailable",
		UserMsg:    "That vehicle isn't available for this trip. Please pick another vehicle.",
	},
	CodePaymentAlreadyRecorded: {
		Code:       CodePaymentAlreadyRecorded,
		HTTPStatus: http.StatusConflict,
		Title:      "Payment Already Recorded",
		UserMsg:    "A payment was already recorded for this invoice.",
	},
	CodeKharchaApprovalDenied: {
		Code:       CodeKharchaApprovalDenied,
		HTTPStatus: http.StatusForbidden,
		Title:      "Approval Not Allowed",
		UserMsg:    "You don't have permission to approve this expense. Ask someone with approval rights.",
	},
	CodeKharchaAlreadyApproved: {
		Code:       CodeKharchaAlreadyApproved,
		HTTPStatus: http.StatusConflict,
		Title:      "Already Approved",
		UserMsg:    "This expense entry has already been approved.",
	},
	CodeFileTooLarge: {
		Code:       CodeFileTooLarge,
		HTTPStatus: http.StatusRequestEntityTooLarge,
		Title:      "File Too Large",
		UserMsg:    "The file is too large. Please upload a file under 5MB.",
	},
}
