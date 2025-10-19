package domain

// Reservation represents a booking request associated with a restaurant table.
type Reservation struct {
    ID               string
    RestaurantID     string
    SectionID        string
    TableID          string
    Status           ReservationStatus
    ReservationDate  string
    ReservationTime  string
    Guests           int
    CustomerName     string
    CustomerPhone    string
    CustomerComments string
}

// ReservationList aggregates reservations with pagination metadata.
type ReservationList struct {
    Items []Reservation
    Total int
}

// NormalizeReservation constructs a Reservation from a loosely typed map.
func NormalizeReservation(raw map[string]any) (Reservation, bool) {
    id := asString(raw["id"])
    if id == "" {
        return Reservation{}, false
    }

    reservation := Reservation{
        ID:               id,
        RestaurantID:     asString(raw["restaurantId"]),
        SectionID:        asString(raw["sectionId"]),
        TableID:          asString(raw["tableId"]),
        ReservationDate:  asString(raw["reservationDate"]),
        ReservationTime:  asString(raw["reservationTime"]),
        Guests:           asInt(raw["numberOfGuests"]),
        CustomerName:     asString(raw["customerName"]),
        CustomerPhone:    asString(raw["customerPhone"]),
        CustomerComments: asString(raw["comments"]),
    }

    status := NormalizeReservationStatus(raw["status"])
    if status == ReservationStatusUnknown {
        status = NormalizeReservationStatus(raw["state"])
    }
    reservation.Status = status

    return reservation, true
}

// BuildReservationList projects payloads into a ReservationList.
func BuildReservationList(payload any) (*ReservationList, bool) {
    container := mapFromPayload(payload)
    if len(container) == 0 {
        return nil, false
    }

    rawItems := asInterfaceSlice(container["items"])
    if len(rawItems) == 0 {
        rawItems = asInterfaceSlice(container["reservations"])
    }
    if len(rawItems) == 0 {
        return nil, false
    }

    result := &ReservationList{Items: make([]Reservation, 0, len(rawItems))}
    for _, item := range rawItems {
        if rawMap, ok := item.(map[string]any); ok {
            if reservation, ok := NormalizeReservation(rawMap); ok {
                result.Items = append(result.Items, reservation)
            }
        }
    }
    if len(result.Items) == 0 {
        return nil, false
    }

    if total := asInt(container["total"]); total > 0 {
        result.Total = total
    } else {
        result.Total = len(result.Items)
    }

    return result, true
}

// BuildReservationDetail extracts a single reservation projection from the payload.
func BuildReservationDetail(payload any) (*Reservation, bool) {
    container := mapFromPayload(payload)
    if len(container) == 0 {
        return nil, false
    }

    if nested, ok := container["reservation"].(map[string]any); ok {
        container = nested
    }

    reservation, ok := NormalizeReservation(container)
    if !ok {
        return nil, false
    }
    return &reservation, true
}
