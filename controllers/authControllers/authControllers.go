package authcontrollers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	jwtvalidation "github.com/akshay0074700747/projectandCompany_management_api-gateway/jwtValidation"
	"github.com/akshay0074700747/projectandCompany_management_protofiles/pb/authpb"
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

func (auth *AuthCtl) loginUser(w http.ResponseWriter, r *http.Request) {

	if cookie, _ := r.Cookie("authentication"); cookie != nil {
		http.Error(w, "you are already logged in...", http.StatusForbidden)
		return
	}

	var req authpb.LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.PrintErr(err, "cannot decode json on login")
		http.Error(w, "error on parsing", http.StatusInternalServerError)
		return
	}

	res, err := auth.Conn.LoginUser(r.Context(), &req)
	if err != nil {
		helpers.PrintErr(err, "error on loginUser")
		http.Error(w, "error on login", http.StatusInternalServerError)
		return
	}

	jsondta, err := json.Marshal(res)
	if err != nil {
		helpers.PrintErr(err, "error on marshaling to json on login")
		http.Error(w, "error on marshling to json", http.StatusInternalServerError)
		return
	}

	cookieString, err := jwtvalidation.GenerateJwt(res.UserID, res.IsAdmin, []byte(secret))
	if err != nil {
		http.Error(w, "there is a problem with loggin in please try again later", http.StatusInternalServerError)
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

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func (auth *AuthCtl) logoutUser(w http.ResponseWriter, r *http.Request) {

	if cookie, _ := r.Cookie("authentication"); cookie == nil {
		http.Error(w, "allready logged out", http.StatusBadRequest)
		return
	}

	cookie := &http.Cookie{
		Name:     "authentication",
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusOK)

	w.Write([]byte("Logged out Successfully..."))
}
