package domain

// SectionSnapshot holds the full state returned by the REST API.
// Payload remains untyped so the adapter can forward whatever structure Nest emits.
type SectionSnapshot struct {
	Payload any
}
