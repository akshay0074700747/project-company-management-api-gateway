package clicontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		helpers.PrintErr(err, "the secret cannot be retrieved...")
	}
	accessKey = os.Getenv("accesskey")
	secretAccessKey = os.Getenv("secretaccess")
}

var (
	accessKey       string
	secretAccessKey string
)

func downloadCli(w http.ResponseWriter, r *http.Request) {

	var res = make(map[string]string)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		helpers.PrintErr(err, "error on unmarshaling to json on downloadcli")
		http.Error(w, "error on unmarshling to json", http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(body, &res); err != nil {
		helpers.PrintErr(err, "error on unmarshaling to json on downloadcli")
		http.Error(w, "error on unmarshling to map", http.StatusInternalServerError)
		return
	}

	pattern := regexp.MustCompile(`\.tar\.gz$`)

	if res["format"] == "" || !pattern.MatchString(res["format"]) {
		helpers.PrintErr(err, "error on unmarshaling to json on downloadcli")
		http.Error(w, "please provide a valid format", http.StatusBadRequest)
		return
	}

	keyName := res["format"]

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretAccessKey, "")),
	)
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		os.Exit(1)
	}

	client := s3.NewFromConfig(cfg)

	bucketName := "project-and-company-management-cli-utitlity-tracker"

	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    &keyName,
	})
	if err != nil {
		fmt.Println("Error listing objects:", err)
		http.Error(w, "Error downloading file from S3", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", res["format"]))

	if _, err = io.Copy(w, resp.Body); err != nil {
		http.Error(w, "Error writing file content to response", http.StatusInternalServerError)
		return
	}

}

func getDownloads(w http.ResponseWriter, r *http.Request) {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretAccessKey, "")),
	)
	if err != nil {
		http.Error(w, "there is a problem", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot congig aws")
		return
	}

	client := s3.NewFromConfig(cfg)

	bucketName := "project-and-company-management-cli-utitlity-tracker"

	resp, err := client.ListObjects(context.TODO(), &s3.ListObjectsInput{
		Bucket: &bucketName,
		Prefix: aws.String("releases/"),
	})
	if err != nil {
		http.Error(w, "there is a problem", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot list objects")
		return
	}

	pattern := regexp.MustCompile(`\.tar\.gz$`)

	var responce []string
	for _, obj := range resp.Contents {
		if *obj.Key != "" && pattern.MatchString(*obj.Key) {
			responce = append(responce, *obj.Key)
		}
	}

	jsondta, err := json.Marshal(responce)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getdownloads")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)

}

//s3://project-and-company-management-cli-utitlity-tracker/releases/
