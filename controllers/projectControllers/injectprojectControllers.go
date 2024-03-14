package projectcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/projectpb"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type ProjectCtl struct {
	Conn           projectpb.ProjectServiceClient
	TaskAssignator *TaskProducer
	Secret         string
}

type TaskProducer struct {
	Producer     *kafka.Producer
	Topic        string
	DeliveryChan chan kafka.Event
}

func NewTaskProducer(topic string) *TaskProducer {

	configMap := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "taskAssignation-producer",
		"acks":              "all",
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		helpers.PrintErr(err, "errror at creating porducer")
		return nil
	}

	deliveryChan := make(chan kafka.Event)

	return &TaskProducer{
		Producer:     producer,
		Topic:        topic,
		DeliveryChan: deliveryChan,
	}
}

func NewProjectCtl(conn *grpc.ClientConn, taskAssignator *TaskProducer, secret string) *ProjectCtl {
	return &ProjectCtl{
		Conn:           projectpb.NewProjectServiceClient(conn),
		TaskAssignator: taskAssignator,
		Secret:         secret,
	}
}

func (proj *ProjectCtl) InjectProjectControllers(r *chi.Mux) {
	r.Post("/project/create", middleware.ValidationMiddlewareClients(proj.createProject))
	r.Post("/project/members/add", middleware.ValidationMiddlewareClients(proj.addMembers))
	r.Get("/project/invites", middleware.ValidationMiddlewareClients(proj.projectInvites))
	r.Post("/project/invites/accept", middleware.ValidationMiddlewareClients(proj.acceptProjectInvite))
	r.Post("/project/members/tasks/assign", middleware.ValidationMiddlewareClients(proj.assignTasks))
	r.Get("/project/details", middleware.ValidationMiddlewareClients(proj.projectDetails))
	r.Get("/project/members", middleware.ValidationMiddlewareClients(proj.getProjectMembers))
	r.Post("/project/login", middleware.ValidationMiddlewareClients(proj.LogintoProject))
	r.Post("/project/member/statu/post", middleware.ValidationMiddlewareAdmins(proj.addMemberStatus))
	r.Get("/project/tasks/get", middleware.ValidationMiddlewareClients(proj.getAssignedTasks))
	r.Get("/project/tasks/download", proj.downloadTask)
	r.Get("/project/members/progress", middleware.ValidationMiddlewareClients(proj.getProgressofMembers))
	r.Get("/project/member/progress", middleware.ValidationMiddlewareClients(proj.getMemberProgress))
	r.Get("/project/progress", middleware.ValidationMiddlewareClients(proj.getProjectProgress))
	r.Post("/project/progress/non-technical/post", middleware.ValidationMiddlewareClients(proj.markProgressofNonTechnical))
	r.Post("/project/task/statuses/post", middleware.ValidationMiddlewareAdmins(proj.addTaskStatuses))
	r.Get("/company/projects/live", middleware.ValidationMiddlewareClients(proj.getLiveProjectsofCompany))
	r.Post("/project/logout", middleware.ValidationMiddlewareClients(proj.logoutFromProject))
	r.Get("/project/members/completed", middleware.ValidationMiddlewareClients(proj.getCompletedMembers))
	r.Get("/project/members/critical", middleware.ValidationMiddlewareClients(proj.getCriticalMembers))
	r.Post("/project/issue/raise", middleware.ValidationMiddlewareClients(proj.raiseIssue))
	r.Get("/project/member/issues", middleware.ValidationMiddlewareClients(proj.getIssuesofMember))
	r.Get("/project/issues", middleware.ValidationMiddlewareClients(proj.getIssues))
	r.Post("/project/task/rate", middleware.ValidationMiddlewareClients(proj.rateTask))
	r.Get("/project/task/feedback", middleware.ValidationMiddlewareClients(proj.feedbackforTask))
	r.Post("/project/task/deadline", middleware.ValidationMiddlewareClients(proj.requestforDeadlineExtension))
	r.Get("/projects/task/extensions", middleware.ValidationMiddlewareClients(proj.getExtensionRequests))
	r.Post("/project/task/extensions/grant", middleware.ValidationMiddlewareClients(proj.grantExtension))
	r.Post("/project/task/verify", middleware.ValidationMiddlewareClients(proj.verifyTaskCompletion))
	r.Get("/project/verify/tasks", middleware.ValidationMiddlewareClients(proj.getverifiedTasks))
	r.Delete("/project/drop",middleware.ValidationMiddlewareClients(proj.dropProject))
	r.Post("/project/members/terminate",middleware.ValidationMiddlewareClients(proj.terminateProjectmembers))
	r.Patch("/project/details/update",middleware.ValidationMiddlewareClients(proj.updateProjectDetails))
	r.Patch("/project/member/edit",middleware.ValidationMiddlewareClients(proj.EditMember))
	r.Patch("/project/feedback/edit",middleware.ValidationMiddlewareClients(proj.editFeedback))
	r.Delete("/project/feedback/delete",middleware.ValidationMiddlewareClients(proj.deleteFeedback))
}
