package service_catalog

import "errors"

var (
	ErrServiceNotFound   = errors.New("service not found")
	ErrServiceDuplicate  = errors.New("service already exists")
	ErrPodNotFound       = errors.New("pod workload not found")
	ErrVMNotFound        = errors.New("vm workload not found")
	ErrInvalidPlatform   = errors.New("platform must be 'kubernetes' or 'vm'")
	ErrDatacenterInvalid = errors.New("datacenter not found")
)
