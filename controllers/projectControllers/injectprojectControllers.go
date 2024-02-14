package projectcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/projectpb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type ProjectCtl struct {
	Conn projectpb.ProjectServiceClient
}

func NewProjectCtl(conn *grpc.ClientConn) *ProjectCtl {
	return &ProjectCtl{
		Conn: projectpb.NewProjectServiceClient(conn),
	}
}

func (proj*ProjectCtl) InjectProjectControllers(r *chi.Mux) {
	r.Get("/create/project", middleware.ValidationMiddlewareClients(proj.createProject))
}
