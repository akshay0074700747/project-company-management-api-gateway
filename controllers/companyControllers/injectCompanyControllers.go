package companycontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/companypb"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type CompanyCtl struct {
	Conn       companypb.CompanyServiceClient
	Secret     string
	JobApplier *JobProducer
}

type JobProducer struct {
	Producer     *kafka.Producer
	Topic        string
	DeliveryChan chan kafka.Event
}

func NewJobProducer(topic string) *JobProducer {

	configMap := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "jobApplier-producer",
		"acks":              "all",
	}

	producer, err := kafka.NewProducer(configMap)
	if err != nil {
		helpers.PrintErr(err, "errror at creating porducer")
		return nil
	}

	deliveryChan := make(chan kafka.Event)

	return &JobProducer{
		Producer:     producer,
		Topic:        topic,
		DeliveryChan: deliveryChan,
	}
}

func NewCompanyCtl(conn *grpc.ClientConn, secret string, jobApplier *JobProducer) *CompanyCtl {
	return &CompanyCtl{
		Conn:       companypb.NewCompanyServiceClient(conn),
		Secret:     secret,
		JobApplier: jobApplier,
	}
}

func (comp *CompanyCtl) InjectCompanyControllers(r *chi.Mux) {
	r.Post("/company/register", middleware.ValidationMiddlewareClients(comp.registerCompany))
	r.Get("/company/types", middleware.ValidationMiddlewareClients(comp.getCompanyTypes))
	r.Get("/permissions", middleware.ValidationMiddlewareClients(comp.getPermissions))
	r.Post("/company/employees/add", middleware.ValidationMiddlewareClients(comp.addEmployees))
	r.Post("/company/roles/permissions/bind", middleware.ValidationMiddlewareClients(comp.attachRolewithPermissions))
	r.Get("/company/roles/permissions", middleware.ValidationMiddlewareClients(comp.getattachedRoleswithPermissions))
	r.Post("/company/types/add", middleware.ValidationMiddlewareAdmins(comp.addCompanyTypes))
	r.Post("/permissions/add", middleware.ValidationMiddlewareAdmins(comp.addPermissions))
	r.Get("/company/details", middleware.ValidationMiddlewareClients(comp.companyDetails))
	r.Get("/company/employees", middleware.ValidationMiddlewareClients(comp.getCompanyEmployees))
	r.Post("/company/login", middleware.ValidationMiddlewareClients(comp.LogintoCompany))
	r.Post("/company/member/status/post", middleware.ValidationMiddlewareClients(comp.addCompanyMemberStatuses))
	r.Post("/company/member/salary/increment", middleware.ValidationMiddlewareClients(comp.salaryIncrementofEmployees))
	r.Post("/company/role/salary/increment", middleware.ValidationMiddlewareClients(comp.salaryIncrementofRole))
	r.Get("/company/salary/average/role", middleware.ValidationMiddlewareClients(comp.getAverageSalaryPerRole))
	r.Get("/company/salay/leaderboard", middleware.ValidationMiddlewareClients(comp.getSalaryLeaderboard))
	r.Post("/compnay/problem/raise", middleware.ValidationMiddlewareClients(comp.raiseProblem))
	r.Get("/company/problems", middleware.ValidationMiddlewareClients(comp.getProbelmsinaCompany))
	r.Get("/company/profile/views", middleware.ValidationMiddlewareClients(comp.getProfileViewsinaCompany))
	r.Get("/company/visitors/online", middleware.ValidationMiddlewareClients(comp.getOnlineCompanyVisitors))
	r.Post("/company/logout", middleware.ValidationMiddlewareClients(comp.logoutFromCompany))
	r.Post("/company/client/add",middleware.ValidationMiddlewareClients(comp.addCompanyClients))
	r.Post("/company/client/associate/project",middleware.ValidationMiddlewareClients(comp.associateClientwithProject))
	r.Get("/company/projects/past",middleware.ValidationMiddlewareClients(comp.getPastProjects))
	r.Get("/company/clients",middleware.ValidationMiddlewareClients(comp.getClients))
	r.Get("/company/revenue",middleware.ValidationMiddlewareClients(comp.revenueGenerated))
	r.Patch("/company/revenue/update",middleware.ValidationMiddlewareClients(comp.updateRevenueStatus))
	r.Post("/company/policies/attach",middleware.ValidationMiddlewareClients(comp.attachCompanyPolicies))
	r.Patch("/company/employee/payment/status",middleware.ValidationMiddlewareClients(comp.updatePaymentStatus))
	r.Post("/company/problems/assign",middleware.ValidationMiddlewareClients(comp.assignProblem))
	r.Post("/company/problems/resolve",middleware.ValidationMiddlewareClients(comp.resolveProblem))
	r.Get("/company/problems/assigned",middleware.ValidationMiddlewareClients(comp.getAssignedProblems))
	r.Post("/company/leave/apply",middleware.ValidationMiddlewareClients(comp.applyforLeave))
	r.Get("/company/leave/requests",middleware.ValidationMiddlewareClients(comp.getLeaveRequests))
	r.Post("/company/leave/grant",middleware.ValidationMiddlewareClients(comp.decideEmployeeLeave))
	r.Get("/company/leaves",middleware.ValidationMiddlewareClients(comp.getLeaves))
	r.Post("/company/jobs/post",middleware.ValidationMiddlewareClients(comp.postJobs))
	r.Post("/company/jobs/apply", middleware.ValidationMiddlewareClients(comp.applyforJob))
	r.Get("/company/jobs/company",middleware.ValidationMiddlewareClients(comp.getJobsofCompany))
	r.Get("/company/job/applications",middleware.ValidationMiddlewareClients(comp.getJobApplications))
	r.Post("/company/job/application/shortlist",middleware.ValidationMiddlewareClients(comp.shortlistApplications))
	r.Post("/company/job/applications/schedule/interview",middleware.ValidationMiddlewareClients(comp.scheduleInterview))
	r.Get("/company/scheduled/interviews",middleware.ValidationMiddlewareClients(comp.sheduledInterviews))
	r.Get("/company/jobs/applications/id",middleware.ValidationMiddlewareClients(comp.getDetailsofApplicationByID))
	r.Get("/company/interviews/scheduled",middleware.ValidationMiddlewareClients(comp.getScheduledInterviewsofUser))
	r.Post("/company/interview/reschedule",middleware.ValidationMiddlewareClients(comp.rescheduleInterview))
	r.Get("/company/jobs/applications/shortlisted",middleware.ValidationMiddlewareClients(comp.getShortlistedApplications))
	r.Get("/company/jobs",middleware.ValidationMiddlewareClients(comp.getJobs))
	r.Get("/company/job/applications/user",middleware.ValidationMiddlewareClients(comp.getAllapplicationsofUser))
	r.Delete("/company/drop",middleware.ValidationMiddlewareClients(comp.dropCompany))
	r.Patch("/company/details/edit",middleware.ValidationMiddlewareClients(comp.editDetais))
	r.Delete("/company/employees/terminate",middleware.ValidationMiddlewareClients(comp.deleteCompanyEmployee))
	r.Patch("/company/employees/edit",middleware.ValidationMiddlewareClients(comp.editCompanyEmployees))
	r.Delete("/company/problems/delete",middleware.ValidationMiddlewareClients(comp.deleteProblem))
	r.Patch("/company/problems/edit",middleware.ValidationMiddlewareClients(comp.editproblems))
	r.Delete("/company/client/drop",middleware.ValidationMiddlewareClients(comp.dropClient))
	r.Patch("/company/policies/edit",middleware.ValidationMiddlewareClients(comp.editCompanyPolicies))
	r.Delete("/company/jobs/delete",middleware.ValidationMiddlewareClients(comp.deleteJob))
	r.Patch("/company/jobs/edit",middleware.ValidationMiddlewareClients(comp.updateJob))
}
