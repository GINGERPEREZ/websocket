package domain

// SectionSnapshot holds the full state returned by the REST API.
// Payload remains untyped so the adapter can forward whatever structure Nest emits.
// When possible, typed projections are stored in RestaurantList or Restaurant for
// use by higher layers without forcing additional parsing.
type SectionSnapshot struct {
	Payload        any
	RestaurantList *RestaurantList
	Restaurant     *Restaurant
}
