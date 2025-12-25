package storage

// Visibility represents file visibility (public or private).
type Visibility string

const (
	// VisibilityPublic makes files publicly accessible
	VisibilityPublic Visibility = "public"

	// VisibilityPrivate makes files private (requires authentication/signed URLs)
	VisibilityPrivate Visibility = "private"
)

// String returns the string representation of visibility.
func (v Visibility) String() string {
	return string(v)
}

// IsPublic returns true if visibility is public.
func (v Visibility) IsPublic() bool {
	return v == VisibilityPublic
}

// IsPrivate returns true if visibility is private.
func (v Visibility) IsPrivate() bool {
	return v == VisibilityPrivate
}
