package inventory

import "errors"

var (
	// ErrDuplicateDatacenter is returned when attempting to create a datacenter with a name that already exists
	ErrDuplicateDatacenter = errors.New("datacenter with this name already exists")

	// ErrDuplicateServer is returned when attempting to create a server with a hostname that already exists
	ErrDuplicateServer = errors.New("server with this hostname already exists")

	// ErrDuplicateInfrastructureHardware is returned when attempting to create infrastructure hardware with a hostname that already exists
	ErrDuplicateInfrastructureHardware = errors.New("infrastructure hardware with this hostname already exists")

	// ErrDatacenterNotFound is returned when a datacenter cannot be found
	ErrDatacenterNotFound = errors.New("datacenter not found")

	// ErrServerNotFound is returned when a server cannot be found
	ErrServerNotFound = errors.New("server not found")

	// ErrInfrastructureHardwareNotFound is returned when infrastructure hardware cannot be found
	ErrInfrastructureHardwareNotFound = errors.New("infrastructure hardware not found")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidID is returned when an invalid ID is provided
	ErrInvalidID = errors.New("invalid id format")
)

// IsDuplicateError checks if an error is a duplicate error
func IsDuplicateError(err error) bool {
	return errors.Is(err, ErrDuplicateDatacenter) || errors.Is(err, ErrDuplicateServer) ||
		errors.Is(err, ErrDuplicateInfrastructureHardware)
}

// IsNotFoundError checks if an error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrDatacenterNotFound) || errors.Is(err, ErrServerNotFound) ||
		errors.Is(err, ErrInfrastructureHardwareNotFound)
}
