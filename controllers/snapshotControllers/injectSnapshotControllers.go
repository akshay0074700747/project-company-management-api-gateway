package snapshotcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-chi/chi"
)

type SnapshotCtl struct {
	Producer     *kafka.Producer
	Topic        string
	DeliveryChan chan kafka.Event
}

func NewSnapshotCtl(topic string) *SnapshotCtl {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": "host.docker.internal:9092",
		"client.id":         "snapShot-producer",
		"acks":              "all",
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		helpers.PrintErr(err, "errror at creating porducer")
		return nil
	}

	deliveryChan := make(chan kafka.Event)

	return &SnapshotCtl{
		Producer:     producer,
		Topic:        topic,
		DeliveryChan: deliveryChan,
	}
}

func (snap *SnapshotCtl) InjectSnapshotControllers(r *chi.Mux) {
	r.Post("/snapshots/push", snap.pushSnapshots)
	r.Get("/snapshots/pull",snap.pullSnapshot)
}
