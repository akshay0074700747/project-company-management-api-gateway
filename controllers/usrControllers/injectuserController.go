package usrcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/rediss"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/userpb"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type UserCtl struct {
	Conn         userpb.UserServiceClient
	Secret       string
	Cache        *rediss.Cache
	Producer     *kafka.Producer
	Topic        string
	DeliveryChan chan kafka.Event
}

func NewUserserviceClient(conn *grpc.ClientConn, secret string, cache *rediss.Cache, topic string) *UserCtl {

	configMap := &kafka.ConfigMap{
		"bootstrap.servers": "host.docker.internal:9092",
		"client.id":         "email-producer",
		"acks":              "all",
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		helpers.PrintErr(err, "errror at creating porducer")
	}

	deliveryChan := make(chan kafka.Event)

	return &UserCtl{
		Conn:         userpb.NewUserServiceClient(conn),
		Secret:       secret,
		Cache:        cache,
		Topic:        topic,
		Producer:     producer,
		DeliveryChan: deliveryChan,
	}
}

func (usr *UserCtl) InjectUserControllers(r *chi.Mux) {
	r.Post("/signup", usr.signupUser)
	r.Get("/roles", middleware.ValidationMiddlewareClients(usr.getRoles))
	r.Post("/status/set", middleware.ValidationMiddlewareClients(usr.setStatus))
	r.Get("/search/members", middleware.ValidationMiddlewareClients(usr.searchAvlMembers))
	r.Post("/roles/post", middleware.ValidationMiddlewareAdmins(usr.addRoles))
	r.Get("/details", middleware.ValidationMiddlewareClients(usr.showUserDetails))
	r.Patch("/status/edit", middleware.ValidationMiddlewareClients(usr.editStatus))
	r.Patch("/details/update", middleware.ValidationMiddlewareClients(usr.updateDetails))
	r.Get("/subscription/plan", middleware.ValidationMiddlewareClients(usr.getSubscriptionPlans))
	r.Post("/subscription/plan/add", middleware.ValidationMiddlewareAdmins(usr.addSubscription))
	r.Post("/subscription/plan/subscribe", middleware.ValidationMiddlewareClients(usr.subscribe))
	r.Get("/subscriptions", middleware.ValidationMiddlewareClients(usr.getSubscriptions))
	r.Get("/subscription/plan/subscribe/order/pay", usr.pay)
	r.Get("/verify/payment", usr.verifyPayment)
	r.Get("/payment/verified", usr.verifiedPayment)
	r.Get("/payments", middleware.ValidationMiddlewareClients(usr.getAllPayments))
}
