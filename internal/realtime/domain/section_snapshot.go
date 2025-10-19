package domain

// SectionSnapshot holds the full state of a section returned by the REST API.
// It keeps a generic payload (map[string]any) to stay decoupled from Nest DTOs.
type SectionSnapshot struct {
	Section map[string]any
}
