package error

import (
	"errors"
	"fmt"
	"net/http"
	errConstant "user-service/constants/error"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"strings"
)

type ValidationResponse struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

var ErrValidator = map[string]string{}

func ErrValidationResponse(err error) (validationResponse []ValidationResponse) {
	var fieldErrors validator.ValidationErrors

	if errors.As(err, &fieldErrors) {
		for _, err := range fieldErrors {
			switch err.Tag() {
			case "required":
				validationResponse = append(validationResponse, ValidationResponse{
					Field:   err.Field(),
					Message: fmt.Sprintf("%s is required", err.Field()),
				})
			case "email":
				validationResponse = append(validationResponse, ValidationResponse{
					Field:   err.Field(),
					Message: fmt.Sprintf("%s is not a valid email address", err.Field()),
				})
			default:
				errValidator, ok := ErrValidator[err.Tag()]
				if ok {
					count := strings.Count(errValidator, "%s")
					if count == 1 {
						validationResponse = append(validationResponse, ValidationResponse{
							Field:   err.Field(),
							Message: fmt.Sprintf(errValidator, err.Field()),
						})
					} else {
						validationResponse = append(validationResponse, ValidationResponse{
							Field:   err.Field(),
							Message: fmt.Sprintf(errValidator, err.Field(), err.Param()),
						})
					}
				} else {
					validationResponse = append(validationResponse, ValidationResponse{
						Field:   err.Field(),
						Message: fmt.Sprintf("something wrong on %s; %s", err.Field(), err.Param()),
					})
				}
			}
		}
	}

	return validationResponse
}

func WrapError(err error) error {
	logrus.Errorf("error: %v", err)
	return err
}

// HTTPStatus maps a known application error to the HTTP status code that best
// describes it, so handlers don't have to hardcode a single status (e.g. 400)
// for every kind of failure returned by the service layer.
func HTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, errConstant.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, errConstant.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, errConstant.ErrUserNotFound), errors.Is(err, errConstant.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, errConstant.ErrRequestValidation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, errConstant.ErrUserAlreadyExist),
		errors.Is(err, errConstant.ErrEmailAlreadyExist),
		errors.Is(err, errConstant.ErrUserNameExist):
		return http.StatusConflict
	case errors.Is(err, errConstant.ErrTooManyRequests):
		return http.StatusTooManyRequests
	default:
		return http.StatusBadRequest
	}
}
