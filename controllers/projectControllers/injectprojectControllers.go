package projectcontrollers

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/rediss"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/projectpb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type ProjectCtl struct {
	Conn           projectpb.ProjectServiceClient
	TaskAssignator *TaskProducer
	Secret         string
	Cache          *rediss.Cache
}

type TaskProducer struct {
	Producer     sarama.SyncProducer
	DeliveryChan chan<- sarama.Message
	Topic        string
}

func NewTaskProducer(topic string) *TaskProducer {

	// configMap := &kafka.ConfigMap{
	// 	"bootstrap.servers": "host.docker.internal:9092",
	// 	"client.id":         "taskAssignation-producer",
	// 	"acks":              "all",
	// }

	// producer, err := kafka.NewProducer(configMap)
	// if err != nil {
	// 	helpers.PrintErr(err, "errror at creating porducer")
	// 	return nil
	// }

	// deliveryChan := make(chan kafka.Event)

	// return &TaskProducer{
	// 	Producer:     producer,
	// 	Topic:        topic,
	// 	DeliveryChan: deliveryChan,
	// }

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 5
	config.Producer.Retry.Backoff = 50 * time.Millisecond

	var producer sarama.SyncProducer
	var err error
	for i := 0; i < 8; i++ {
		producer, err = sarama.NewSyncProducer([]string{"apache-kafka-service:9092"}, config)
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

	return &TaskProducer{
		Topic:        topic,
		Producer:     producer,
		DeliveryChan: make(chan<- sarama.Message),
	}
}

func NewProjectCtl(conn *grpc.ClientConn, taskAssignator *TaskProducer, secret string, cache *rediss.Cache) *ProjectCtl {
	return &ProjectCtl{
		Conn:           projectpb.NewProjectServiceClient(conn),
		TaskAssignator: taskAssignator,
		Secret:         secret,
		Cache:          cache,
	}
}

func (proj *ProjectCtl) InjectProjectControllers(r *chi.Mux) {   
	r.Post("/project/create", middleware.ValidationMiddlewareClients(proj.createProject))
	r.Post("/project/members/add", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.addMembers)))
	r.Get("/project/invites", middleware.ValidationMiddlewareClients(proj.projectInvites))
	r.Post("/project/invites/accept", middleware.ValidationMiddlewareClients(proj.acceptProjectInvite))
	r.Post("/project/members/tasks/assign", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.assignTasks)))
	r.Get("/project/details", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.projectDetails)))
	r.Get("/project/members", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getProjectMembers)))
	r.Post("/project/login", middleware.ValidationMiddlewareClients(proj.LogintoProject))
	r.Post("/project/member/statu/post", middleware.ValidationMiddlewareAdmins(proj.addMemberStatus))
	r.Get("/project/tasks/get", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getAssignedTasks)))
	r.Get("/project/tasks/download", proj.downloadTask)
	r.Get("/project/members/progress", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getProgressofMembers)))
	r.Get("/project/member/progress", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getMemberProgress)))
	r.Get("/project/progress", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getProjectProgress)))
	r.Post("/project/progress/non-technical/post", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.markProgressofNonTechnical)))
	r.Post("/project/task/statuses/post", middleware.ValidationMiddlewareAdmins(proj.addTaskStatuses))
	r.Get("/company/projects/live", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getLiveProjectsofCompany)))
	r.Post("/project/logout", middleware.ValidationMiddlewareClients(proj.logoutFromProject))
	r.Get("/project/members/completed", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getCompletedMembers)))
	r.Get("/project/members/critical", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getCriticalMembers)))
	r.Post("/project/issue/raise", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.raiseIssue)))
	r.Get("/project/member/issues", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getIssuesofMember)))
	r.Get("/project/issues", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getIssues)))
	r.Post("/project/task/rate", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.rateTask)))
	r.Get("/project/task/feedback", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.feedbackforTask)))
	r.Post("/project/task/deadline", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.requestforDeadlineExtension)))
	r.Get("/projects/task/extensions", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getExtensionRequests)))
	r.Post("/project/task/extensions/grant", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.grantExtension)))
	r.Post("/project/task/verify", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.verifyTaskCompletion)))
	r.Get("/project/verify/tasks", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.getverifiedTasks)))
	r.Delete("/project/drop", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.dropProject)))
	r.Post("/project/members/terminate", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.terminateProjectmembers)))
	r.Patch("/project/details/update", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.updateProjectDetails)))
	r.Patch("/project/member/edit", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.EditMember)))
	r.Patch("/project/feedback/edit", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.editFeedback)))
	r.Delete("/project/feedback/delete", middleware.ValidationMiddlewareClients(proj.ProjectMiddleware(proj.deleteFeedback)))
}
