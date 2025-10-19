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
	if len(brokers) == 0 {
		// No brokers configured; skip starting consumers. In production this should be set
		// by KAFKA_BROKER(S). We avoid calling kafka.NewReader with an empty broker list.
		return
	}
	for _, topic := range topics {
		go func(tp string) {
			consumer := NewKafkaConsumer(brokers, "realtime-group", tp)
			consumer.Consume(ctx, func(msg *domain.Message) error {
				return registry.Dispatch(ctx, msg)
			})
		}(topic)
	}
}
