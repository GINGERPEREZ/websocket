package broker

import (
	"context"
	"mesaYaWs/internal/realtime/domain"
	"mesaYaWs/internal/realtime/infrastructure"
)

func StartKafkaConsumers(
	ctx context.Context,
	registry *infrastructure.HandlerRegistry,
	brokers []string,
	topics []string,
) {
	for _, topic := range topics {
		go func(tp string) {
			consumer := NewKafkaConsumer(brokers, "realtime-group", tp)
			consumer.Consume(ctx, func(msg *domain.Message) error {
				return registry.Dispatch(ctx, msg)
			})
		}(topic)
	}
}
