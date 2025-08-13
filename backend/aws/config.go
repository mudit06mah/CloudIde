package aws

import (
    "context"
    "log"
    "os"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var AwsConfig aws.Config
var s3Client *s3.Client

func InitAWSConfig() {

    region := os.Getenv("AWS_REGION")
    
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
    if err != nil {
        log.Fatalf("failed to load configuration, %v", err)
    }

    AwsConfig = cfg
	s3Client = s3.NewFromConfig(AwsConfig)
	log.Println("AWS configuration initialized successfully")
}
