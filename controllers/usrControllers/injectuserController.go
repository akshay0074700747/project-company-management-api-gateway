package usrcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_api-gateway/middleware"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/userpb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type UserCtl struct {
	Conn   userpb.UserServiceClient
	Secret string
}

func NewUserserviceClient(conn *grpc.ClientConn, secret string) *UserCtl {

	return &UserCtl{
		Conn:   userpb.NewUserServiceClient(conn),
		Secret: secret,
	}
}

func (usr *UserCtl) InjectUserControllers(r *chi.Mux) {
	r.Post("/signup", usr.signupUser)
	r.Get("/roles", middleware.ValidationMiddlewareClients(usr.getRoles))
	r.Post("/status/set", middleware.ValidationMiddlewareClients(usr.setStatus))
	r.Get("/search/members", middleware.ValidationMiddlewareClients(usr.searchAvlMembers))
	r.Post("/roles/post", middleware.ValidationMiddlewareAdmins(usr.addRoles))
	r.Get("/details", middleware.ValidationMiddlewareClients(usr.showUserDetails))
}
