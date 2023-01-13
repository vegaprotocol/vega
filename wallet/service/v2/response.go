package v2

type Response struct {
	// Result is REQUIRED on success. This member MUST NOT exist if there was an
	// error invoking the method.
	Result Result `json:"result,omitempty"`

	// Error is REQUIRED on error. This member MUST NOT exist if there was no
	// error triggered during invocation.
	Error *ErrorDetails `json:"error,omitempty"`
}

// Result is just a nicer way to describe what's expected to be returned by the
// handlers.
type Result interface{}

// ErrorDetails is returned when an HTTP call encounters an error.
type ErrorDetails struct {
	// Message provides a short description of the error.
	// The message SHOULD be limited to a concise single sentence.
	Message string `json:"message"`

	// Data is a primitive or a structured value that contains additional
	// information about the error. This may be omitted.
	// The value of this member is defined by the Server (e.g. detailed error
	// information, nested errors etc.).
	Data string `json:"data,omitempty"`
}
