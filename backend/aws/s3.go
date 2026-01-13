package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DownloadTemplate( projectType string, workspaceId string ) (string, error) {
    fmt.Println("Reached DownloadTemplate with projectType:", projectType)
    var prefix string
    //Determine Prefix based on project type
    switch projectType {
        case "nodejs":
            prefix = "templates/node/"
        case "python":
            prefix = "templates/python/"
        case "golang":
            prefix = "templates/golang/"
        case "cpp":
            prefix = "templates/cpp/"
        default:
            return "", fmt.Errorf("unsupported project type: %s", projectType)
    }
    
    cacheDir := filepath.Join(os.Getenv("CACHE_DIR"),workspaceId)
    bucket := os.Getenv("AWS_S3_BUCKET")
   
    //List folder contents:
    resp, err := s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
        Bucket: &bucket,
        Prefix: &prefix,
    })
    if err != nil {
        return "", fmt.Errorf("failed to list object with prefix %s: %v", prefix, err)
    }

    for _, obj := range resp.Contents {
        key := *obj.Key
        if( key == prefix || strings.HasSuffix(key, "/") ) {
            continue    //Skip directories
        }

        relPath := strings.TrimPrefix(key, prefix)
        localPath := filepath.Join(cacheDir, relPath)

        if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
            return "", fmt.Errorf("failed to create directory %s: %v", filepath.Dir(localPath), err)
        }

        out,err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
            Bucket: &bucket,
            Key:    &key,
        })
        if err != nil{
            return "", fmt.Errorf("failed to get object %s: %v", key, err)
        }
        defer out.Body.Close()

        //Add to cache folder
        file,err := os.Create(localPath)
        if err != nil {
            return "", fmt.Errorf("failed to create file %s: %v", localPath, err)
        }
        defer file.Close()

        _,err = io.Copy(file, out.Body)
        if err != nil {
            return "", fmt.Errorf("failed to copy object %s to file %s: %v", key, localPath, err)
        }
    }

    return cacheDir,nil
    
}