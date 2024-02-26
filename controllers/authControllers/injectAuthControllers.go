package authcontrollers

import (
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/authpb"
	"github.com/go-chi/chi"
	"google.golang.org/grpc"
)

type AuthCtl struct {
	Conn   authpb.AuthServiceClient
	Secret string
}

func NewAuthCtl(conn *grpc.ClientConn, secret string) *AuthCtl {
	return &AuthCtl{
		Conn:   authpb.NewAuthServiceClient(conn),
		Secret: secret,
	}
}

func (auth *AuthCtl) InjectAuthControllers(r *chi.Mux) {
	r.Post("/login", auth.loginUser)
	r.Post("/logout", auth.logoutUser)
}
