package companycontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/companypb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type CompanyCtl struct {
	Conn   companypb.CompanyServiceClient
	Secret string
}

func NewCompanyCtl(conn *grpc.ClientConn, secret string) *CompanyCtl {
	return &CompanyCtl{
		Conn:   companypb.NewCompanyServiceClient(conn),
		Secret: secret,
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
	r.Post("/company/member/status/post",middleware.ValidationMiddlewareClients(comp.addCompanyMemberStatuses))
	r.Post("/company/member/salary/increment",middleware.ValidationMiddlewareClients(comp.salaryIncrementofEmployees))
	r.Post("/company/role/salary/increment",middleware.ValidationMiddlewareClients(comp.salaryIncrementofRole))
	r.Get("/company/salary/average/role",middleware.ValidationMiddlewareClients(comp.getAverageSalaryPerRole))
	r.Get("/company/salay/leaderboard",middleware.ValidationMiddlewareClients(comp.getSalaryLeaderboard))
	r.Post("/compnay/problem/raise",middleware.ValidationMiddlewareClients(comp.raiseProblem))
	r.Get("/company/problems",middleware.ValidationMiddlewareClients(comp.getProbelmsinaCompany))
	r.Get("/company/profile/views",middleware.ValidationMiddlewareClients(comp.getProfileViewsinaCompany))
	r.Get("/company/visitors/online",middleware.ValidationMiddlewareClients(comp.getOnlineCompanyVisitors))
}
