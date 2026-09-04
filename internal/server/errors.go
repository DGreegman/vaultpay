package server

import (
	"errors"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// errorBody is the single, consistent error shape across every endpoint
// (PRD §13.1: one error shape everywhere).
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code	string 				`json:"code"`
	Message	string				`json:"message"`
	Fields	map[string]string	`json:"fields,omitempty"`
}

func writeError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(errorBody{
		Error: errorDetail{Code: code, Message: message},
	})
}

// writeValidationError turns validator failures into a 422 with a
// per-field breakdown, so the client knows exactly what to fix.

func writeValidationError(c *fiber.Ctx, err error) error {
	fields := make(map[string]string)

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		for _, fe := range ve {
			fields[fe.Field()] = fe.Tag()
		}
	}

	return c.Status(fiber.StatusUnprocessableEntity).JSON(errorBody{
		Error: errorDetail{
			Code: "validation_field",
			Message: "one or more fields are invalid",
			Fields: fields,
		},
	})
}

// errorHandler is Fiber's last line of defence: anything a handler returns
// as an error - including Fiber's own 404s and 405s - lands here and leaves
// as our standard error shape.(PRD §13.1: one error shape everywhere).

func errorHandler(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	code := "internal_error"
	message := "something went wrong"

	var fe *fiber.Error 
	if errors.As(err, &fe) {
		status = fe.Code
		code = httpErrorCode(status)
		message = fe.Message
	}else {
		// Not a Fiber error means an unexpected one escape a handler,
		// Log the detail: tell the cleient nothing 
		log.Printf("undexpected error: %s %s: %v", c.Method(), c.Path(), err)
	}
	return writeError(c, status, code, message)
}

// httpErrorCode maps a status machine-readable code, so clients can switch on strings than on numbers
func httpErrorCode(status int) string {
	switch status {
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusRequestEntityTooLarge:
		return "request_too_large"
	default: 
		return "internal_error"
	}
}