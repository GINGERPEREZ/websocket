package app

import "mesaYaWs/internal/realtime/infrastructure"

func NewAppHub() *infrastructure.Hub {
	return infrastructure.NewHub()
}
