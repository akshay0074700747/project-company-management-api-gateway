package companycontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/companypb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type CompanyCtl struct {
	Conn companypb.CompanyServiceClient
}

func NewCompanyCtl(conn *grpc.ClientConn) *CompanyCtl {
	return &CompanyCtl{
		Conn: companypb.NewCompanyServiceClient(conn),
	}
}

func (comp *CompanyCtl) InjectCompanyControllers(r *chi.Mux) {
	r.Post("/company/register", middleware.ValidationMiddlewareClients(comp.registerCompany))
}
