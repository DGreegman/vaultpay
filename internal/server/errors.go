package server

import (
	"errors"

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