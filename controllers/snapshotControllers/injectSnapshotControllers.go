package snapshotcontrollers

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/go-chi/chi"
)

type SnapshotCtl struct {
	Producer     sarama.SyncProducer
	DeliveryChan chan<- sarama.Message
	Topic        string
}

func NewSnapshotCtl(topic string) *SnapshotCtl {
// 	configMap := &kafka.ConfigMap{
// 		"bootstrap.servers": "host.docker.internal:9092",
// 		"client.id":         "snapShot-producer",
// 		"acks":              "all",
// 	}

// 	producer, err := kafka.NewProducer(configMap)
// 	if err != nil {
// 		helpers.PrintErr(err, "errror at creating porducer")
// 		return nil
// 	}

// 	deliveryChan := make(chan kafka.Event)

// 	return &SnapshotCtl{
// 		Producer:     producer,
// 		Topic:        topic,
// 		DeliveryChan: deliveryChan,
// 	}

config := sarama.NewConfig()
config.Producer.RequiredAcks = sarama.WaitForAll
config.Producer.Return.Successes = true
config.Producer.Retry.Max = 5
config.Producer.Retry.Backoff = 50 * time.Millisecond

producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
if err != nil {
	helpers.PrintErr(err, "error happeed at creating producer")
}

return &SnapshotCtl{
	Topic:        topic,
	Producer:     producer,
	DeliveryChan: make(chan<- sarama.Message),
}
}

func (snap *SnapshotCtl) InjectSnapshotControllers(r *chi.Mux) {
	r.Post("/snapshots/push", snap.pushSnapshots)
	r.Get("/snapshots/pull",snap.pullSnapshot)
}
