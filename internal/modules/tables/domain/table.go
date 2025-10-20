package domain

import "mesaYaWs/internal/shared/normalization"

// Table represents a seating resource within a section.
type Table struct {
	ID           string
	SectionID    string
	Number       int
	Capacity     int
	State        TableState
	PosX         float64
	PosY         float64
	Width        float64
	TableImageID int
	ChairImageID int
}

// TableList contains a collection of tables alongside pagination metadata.
type TableList struct {
	Items []Table
	Total int
}

// NormalizeTable attempts to construct a Table from an arbitrary map payload.
func NormalizeTable(raw map[string]any) (Table, bool) {
	id := normalization.AsString(raw["id"])
	if id == "" {
		return Table{}, false
	}
	table := Table{
		ID:           id,
	SectionID:    normalization.AsString(raw["sectionId"]),
	Number:       normalization.AsInt(raw["number"]),
	Capacity:     normalization.AsInt(raw["capacity"]),
	PosX:         normalization.AsFloat64(raw["posX"]),
	PosY:         normalization.AsFloat64(raw["posY"]),
	Width:        normalization.AsFloat64(raw["width"]),
	TableImageID: normalization.AsInt(raw["tableImageId"]),
	ChairImageID: normalization.AsInt(raw["chairImageId"]),
	}

	state := NormalizeTableState(raw["state"])
	if state == TableStateUnknown {
		state = NormalizeTableState(raw["status"])
	}
	table.State = state

	return table, true
}

// BuildTableList tries to project the payload into a TableList structure.
func BuildTableList(payload any) (*TableList, bool) {
	container := normalization.MapFromPayload(payload)
	if len(container) == 0 {
		return nil, false
	}

	rawItems := normalization.AsInterfaceSlice(container["items"])
	if len(rawItems) == 0 {
	rawItems = normalization.AsInterfaceSlice(container["tables"])
	}
	if len(rawItems) == 0 {
		return nil, false
	}

	result := &TableList{Items: make([]Table, 0, len(rawItems))}
	for _, item := range rawItems {
		if rawMap, ok := item.(map[string]any); ok {
			if table, ok := NormalizeTable(rawMap); ok {
				result.Items = append(result.Items, table)
			}
		}
	}
	if len(result.Items) == 0 {
		return nil, false
	}

	if total := normalization.AsInt(container["total"]); total > 0 {
		result.Total = total
	} else {
		result.Total = len(result.Items)
	}

	return result, true
}

// BuildTableDetail attempts to extract a single table from the payload.
func BuildTableDetail(payload any) (*Table, bool) {
	container := normalization.MapFromPayload(payload)
	if len(container) == 0 {
		return nil, false
	}

	if nested, ok := container["table"].(map[string]any); ok {
		container = nested
	}

	table, ok := NormalizeTable(container)
	if !ok {
		return nil, false
	}
	return &table, true
}
