package middleware

import (
	"context"
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

		defer func() {
			if r := recover(); r != nil {
				http.Error(w, "You do not have the authority to do this operation", http.StatusBadRequest)
				return
			}
		}()

		cookie, err := r.Cookie("authentication")
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
			http.Error(w, "this route can be only accessed by admins", http.StatusBadRequest)
			return
		}

		next(w, r)
	}

}

func ValidationMiddlewareClients(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		// defer func() {
		// 	if r := recover(); r != nil {
		// 		http.Error(w, "You do not have the authority to do this operation", http.StatusBadRequest)
		// 		return
		// 	}
		// }()
		var ctx = r.Context()

		cookie, err := r.Cookie("authentication")
		if err != nil {
			http.Error(w, "cookie not found , please login...", http.StatusBadRequest)
			helpers.PrintErr(err, "cookie is not found...")
			return
		}

		projCookie, _ := r.Cookie("projectCookie")
		compCookie, _ := r.Cookie("companyCookie")

		if projCookie != nil {
			projcookieVal := projCookie.Value
			projclaimsMap, err := jwtvalidation.ValidateTokenforProject(projcookieVal, []byte(secret))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				helpers.PrintErr(err, "not logged into the project")
				return
			}
			ctx = context.WithValue(ctx, "projectBool", true)
			ctx = context.WithValue(ctx, "projectID", projclaimsMap["projectID"].(string))
			ctx = context.WithValue(ctx, "projectRole", projclaimsMap["role"].(string))
			ctx = context.WithValue(ctx, "projectPermission", projclaimsMap["permission"].(string))
		}
		if compCookie != nil {
			compcookieVal := compCookie.Value
			compclaimsMap, err := jwtvalidation.ValidateTokenforCompany(compcookieVal, []byte(secret))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				helpers.PrintErr(err, "not logged into the company")
				return
			}
			ctx = context.WithValue(ctx, "companyBool", true)
			ctx = context.WithValue(ctx, "companyID", compclaimsMap["companyID"].(string))
			ctx = context.WithValue(ctx, "companyRole", compclaimsMap["role"].(string))
			ctx = context.WithValue(ctx, "companyPermission", compclaimsMap["permission"].(string))
		}

		cookieVal := cookie.Value

		claimsMap, err := jwtvalidation.ValidateToken(cookieVal, []byte(secret))
		if err != nil {
			http.Error(w, "there has been an error in getting the validating the cookie", http.StatusInternalServerError)
			helpers.PrintErr(err, "cookie cannot be found")
			return
		}

		userID := claimsMap["userID"].(string)
		if userID == "" {
			http.Error(w, "the userID is not valid", http.StatusBadRequest)
			helpers.PrintErr(err, "userID is not valid")
			return
		}

		ctx = context.WithValue(ctx, "userID", userID)
		next(w, r.WithContext(ctx))
	}

}
