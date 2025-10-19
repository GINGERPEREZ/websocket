package domain

import "strings"

// TableState represents realtime availability for a table within a section layout.
type TableState string

const (
	TableStateUnknown   TableState = ""
	TableStateAvailable TableState = "AVAILABLE"
	TableStateReserved  TableState = "RESERVED"
	TableStateSeated    TableState = "SEATED"
	TableStateBlocked   TableState = "BLOCKED"
	TableStateCleaning  TableState = "CLEANING"
)

var allowedTableStates = map[string]TableState{
	string(TableStateAvailable): TableStateAvailable,
	string(TableStateReserved):  TableStateReserved,
	string(TableStateSeated):    TableStateSeated,
	string(TableStateBlocked):   TableStateBlocked,
	string(TableStateCleaning):  TableStateCleaning,
}

// NormalizeTableState coerces any input into a canonical table state.
// Unknown values are uppercased and returned to preserve compatibility with upstream.
func NormalizeTableState(value any) TableState {
	s, ok := value.(string)
	if !ok {
		return TableStateUnknown
	}
	trimmed := strings.ToUpper(strings.TrimSpace(s))
	if trimmed == "" {
		return TableStateUnknown
	}
	if state, ok := allowedTableStates[trimmed]; ok {
		return state
	}
	return TableState(trimmed)
}
