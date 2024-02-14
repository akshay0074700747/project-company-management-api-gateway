package usrcontrollers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	jwtvalidation "github.com/akshay0074700747/projectandCompany_management_api-gateway/jwtValidation"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/userpb"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		helpers.PrintErr(err, "the secret cannot be retrieved...")
	}
	secret = os.Getenv("secret")
}

var (
	secret string
)

func (usr *UserCtl) signupUser(w http.ResponseWriter, r *http.Request) {

	if cookie, _ := r.Cookie("authentication"); cookie != nil {
		http.Error(w, "you are already logged in...", http.StatusForbidden)
		return
	}

	var req userpb.SignupUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot parse the signup req")
	}

	res, err := usr.Conn.Signupuser(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "there is a problem withb signupUser")
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on signup")
		return
	}

	cookieString, err := jwtvalidation.GenerateJwt(res.Id, false, []byte(secret))
	if err != nil {
		http.Error(w, "there is a problem with signing up in please try again later", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot create jwt")
		return
	}

	cookie := &http.Cookie{
		Name:     "authentication",
		Value:    cookieString,
		Expires:  time.Now().Add(48 * time.Hour),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusCreated)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}
