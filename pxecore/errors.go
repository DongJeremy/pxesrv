package pxecore

import "fmt"

// PXEError is an error type returned upon runtime errors.
type PXEError struct {
	err error
}

// PXEErrorFromString returns a PXEError from the given error string.
func PXEErrorFromString(format string, args ...interface{}) *PXEError {
	return &PXEError{
		err: fmt.Errorf(format, args...),
	}
}

// PXEErrorFromError returns a PXEError from the given error object.
func PXEErrorFromError(err error) *PXEError {
	return &PXEError{
		err: err,
	}
}

func (ce PXEError) Error() string {
	return fmt.Sprintf("error occur: %v", ce.err)
}
