package clicontroller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/akshay0074700747/projectandCompany_management_api-gateway/helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
)

type CliCtl struct {
	Client *s3.S3
	BucketName string
}

func init() {
	if err := godotenv.Load(".env"); err != nil {
		helpers.PrintErr(err, "the secret cannot be retrieved...")
	}
	accessKey = os.Getenv("accesskey")
	secretAccessKey = os.Getenv("secretaccess")
}

type GetDownloads struct {
	Key  string `json:"Key"`
	Size string `json:"Size"`
}

var (
	accessKey       string
	secretAccessKey string
	converter       float32
)

func getDownloads(w http.ResponseWriter, r *http.Request) {

	creds := credentials.NewStaticCredentials(accessKey, secretAccessKey, "")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("ap-south-1"),
		Credentials: creds,
	})
	if err != nil {
		fmt.Println("Error creating session:", err)
		return
	}

	svc := s3.New(sess)

	bucketName := "project-and-company-management-cli-utitlity-tracker"

	// Create a ListObjects input struct
	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	}

	var result []GetDownloads

	if err := svc.ListObjectsPages(input, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		converter = 1048576
		for _, object := range page.Contents {
			sizeMB := float32(*object.Size) / converter
			result = append(result, GetDownloads{
				Key:  *object.Key,
				Size: fmt.Sprintf("%f MB", sizeMB),
			})

		}
		return true
	}); err != nil {
		fmt.Println("Error listing objects:", err)
		return
	}

	jsondta, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "there is a problem with parsing to json", http.StatusInternalServerError)
		helpers.PrintErr(err, "cannot parse to json on getdownloads")
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsondta)
}

func downloadCli(w http.ResponseWriter, r *http.Request)  {
	
}