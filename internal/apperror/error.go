package apperror

type PublicError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *PublicError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return e.Message
}

func (e *PublicError) Unwrap() error {
	return e.Err
}

func NewPublicError(status int, code string, message string) *PublicError {
	return &PublicError{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func WrapPublicError(status int, code string, message string, err error) *PublicError {
	return &PublicError{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

type FieldValidationError struct {
	Fields map[string]string
}

func (e *FieldValidationError) Error() string {
	return "validation failed"
}

func NewFieldValidationError(fields map[string]string) *FieldValidationError {
	return &FieldValidationError{
		Fields: fields,
	}
}
