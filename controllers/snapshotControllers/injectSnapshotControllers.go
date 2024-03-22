package snapshotcontrollers

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
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

	var producer sarama.SyncProducer
	var err error
	for i := 0; i < 8; i++ {
		producer, err = sarama.NewSyncProducer([]string{"host.docker.internal:29092"}, config)
		if err != nil {
			if i == 7 {
				log.Fatal("Closingg: %v", err)
			}
			fmt.Println("Error creating producer : ", i, ": %v", err)
			time.Sleep(time.Second * 3)
		} else {
			break
		}
	}

	return &SnapshotCtl{
		Topic:        topic,
		Producer:     producer,
		DeliveryChan: make(chan<- sarama.Message),
	}
}

func (snap *SnapshotCtl) InjectSnapshotControllers(r *chi.Mux) {
	r.Post("/snapshots/push", snap.pushSnapshots)
	r.Get("/snapshots/pull", snap.pullSnapshot)
}
