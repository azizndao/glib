package errs

// Helper functions for creating common error types

// NewBadRequest creates a new BadRequest error builder
func NewBadRequest() *Builder {
	return B().Code(InvalidArgument)
}

// NewUnauthorized creates a new Unauthenticated error builder
func NewUnauthorized() *Builder {
	return B().Code(Unauthenticated)
}

// NewForbidden creates a new PermissionDenied error builder
func NewForbidden() *Builder {
	return B().Code(PermissionDenied)
}

// NewNotFound creates a new NotFound error builder
func NewNotFound() *Builder {
	return B().Code(NotFound)
}

// NewConflict creates a new AlreadyExists error builder
func NewConflict() *Builder {
	return B().Code(AlreadyExists)
}

// NewInternal creates a new Internal error builder
func NewInternal() *Builder {
	return B().Code(Internal)
}

// WithMessage sets the message on the builder and returns the error
func (b *Builder) WithMessage(msg string) error {
	return b.Msg(msg).Err()
}
