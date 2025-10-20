package domain

import (
	reservations "mesaYaWs/internal/modules/reservations/domain"
	restaurants "mesaYaWs/internal/modules/restaurants/domain"
	tables "mesaYaWs/internal/modules/tables/domain"
)

// SectionSnapshot holds the full state returned by the REST API.
// Payload remains untyped so the adapter can forward whatever structure Nest emits.
// When possible, typed projections are stored for reuse by higher layers without
// forcing additional parsing.
type SectionSnapshot struct {
	Payload         any
	RestaurantList  *restaurants.RestaurantList
	Restaurant      *restaurants.Restaurant
	TableList       *tables.TableList
	Table           *tables.Table
	ReservationList *reservations.ReservationList
	Reservation     *reservations.Reservation
}
