package middleware

import (
	"net/http"
	"os"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	jwtvalidation "github.com/akshay0074700747/projectandCompany_management_api-gateway/jwtValidation"
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

func ValidationMiddlewareAdmins(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("authorization")
		if err != nil {
			http.Error(w, "cookie not found , please login...", http.StatusBadRequest)
			helpers.PrintErr(err, "cookie is not found...")
			return
		}

		cookieVal := cookie.Value

		claimsMap, err := jwtvalidation.ValidateToken(cookieVal, []byte(secret))
		if err != nil {
			http.Error(w, "there has been an error in getting the validating the cookie", http.StatusInternalServerError)
			helpers.PrintErr(err, "cookie cannot be found")
			return
		}

		if !claimsMap["isadmin"].(bool) {
			http.Error(w,"this route can be only accessed by admins",http.StatusBadRequest)
			return
		}

		next(w, r)
	}

}

func ValidationMiddlewareClients(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		cookie, err := r.Cookie("authorization")
		if err != nil {
			http.Error(w, "cookie not found , please login...", http.StatusBadRequest)
			helpers.PrintErr(err, "cookie is not found...")
			return
		}

		cookieVal := cookie.Value

		claimsMap, err := jwtvalidation.ValidateToken(cookieVal, []byte(secret))
		if err != nil {
			http.Error(w, "there has been an error in getting the validating the cookie", http.StatusInternalServerError)
			helpers.PrintErr(err, "cookie cannot be found")
			return
		}

		if claimsMap["userID"].(string) == "" {
			http.Error(w, "the userID is not valid", http.StatusBadRequest)
			helpers.PrintErr(err, "userID is not valid")
			return
		}
		next(w, r)
	}

}
