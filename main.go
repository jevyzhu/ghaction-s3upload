package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	maxPartSize   = int64(5 * 1024 * 1024)
	maxRetries    = 0
	retryInterval = 15 * time.Second
)

type UploadPartT struct {
	completedPart types.CompletedPart
	partNumber    int
	outptut       *s3.CreateMultipartUploadOutput
}

type FileInfoT struct {
	filePath string
	size     int64
}

func main() {
	log.SetOutput(os.Stdout)
	d := flag.String("p", ".", "path to upload")
	s3Path := flag.String("s", "cf-buildpacks-bucket/test", "s3 path to upload")
	maxPartsCurr := flag.Int("m", 10, "max upload concurrency")
	flag.Parse()
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("FAILED to load SDK configuration, %v", err)
		return
	}
	var filelist []FileInfoT
	filelist, err = getFileList(*d)
	if err != nil {
		log.Fatalf("FAILED to get files, %v", err)
		return
	}
	client := s3.NewFromConfig(cfg)
	failList := []string{}
	buffChan := make(chan int, *maxPartsCurr)
	for _, f := range filelist {
		resp, partCh, err := uploadInParts(client, *s3Path, f, buffChan)
		if err != nil {
			failList = append(failList, f.filePath)
			if err := abortMultipartUpload(client, resp); err != nil {
				log.Printf("[%s]: FAILED abort: %v", f.filePath, err)
			}
			continue
		}
		if mergeAllPars(client, f, partCh, resp) != nil {
			failList = append(failList, f.filePath)
		}
	}
	for _, f := range failList {
		log.Printf("FAILED UPLOAD: %s", f)
	}
}

func uploadInParts(s3Client *s3.Client, s3Path string, fchan FileInfoT, buffChan chan int) (*s3.CreateMultipartUploadOutput, []UploadPartT, error) {
	file, err := os.Open(fchan.filePath)
	if err != nil {
		log.Printf("[%s] ERROR: uploading - %v", fchan.filePath, err)
		return nil, nil, err
	}
	defer file.Close()
	tmpPaths := strings.SplitN(s3Path, "/", 2)
	bucketName := tmpPaths[0]
	objPath := "/"
	if len(tmpPaths) == 2 {
		objPath = tmpPaths[1]
	}
	path := filepath.Join(objPath, filepath.Base(fchan.filePath))
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(path),
		ContentType: aws.String("application/octet-stream"),
	}
	resp, err := s3Client.CreateMultipartUpload(context.TODO(), input)
	if err != nil {
		log.Printf("[%s]: S3 ERROR: %v", fchan.filePath, err)
		return nil, nil, err
	}
	log.Printf("[%s]: created multipart-upload request", fchan.filePath)

	chanSize := (fchan.size + maxPartSize - 1) / maxPartSize
	partNumber := 1
	buffer := make([]byte, maxPartSize)
	partChan := make([]UploadPartT, chanSize)
	wg := sync.WaitGroup{}
	quitChan := make(chan error)
	for n, err := file.Read(buffer); err == nil && n != 0; {
		wg.Add(1)
		go func(d []byte, partN int) {
			defer wg.Done()
			completedPart, err := uploadPartData(s3Client, resp, d, partN, fchan.filePath, buffChan)
			if err != nil {
				log.Printf("[%s]: ERROR upoload #%v - %v", fchan.filePath, partN, err)
				quitChan <- err
			} else {
				partChan[partN-1] = UploadPartT{completedPart, partN, resp}
			}
		}(buffer[0:n], partNumber)
		partNumber++
		n, err = file.Read(buffer)
	}
	go func() {
		wg.Wait()
		quitChan <- nil
	}()
	for {
		select {
		case e := <-quitChan:
			return resp, partChan, e
		}
	}
}

func mergeAllPars(s3Client *s3.Client, f FileInfoT, partChan []UploadPartT, resp *s3.CreateMultipartUploadOutput) error {
	parts := []types.CompletedPart{}
	for _, part := range partChan {
		parts = append(parts, part.completedPart)
	}
	tryNum := 0
	for tryNum <= maxRetries {
		if r, err := finishUpload(s3Client, resp, parts); err == nil {
			log.Printf("[%s]: successfully upload -> %s", f.filePath, *r.Location)
			break
		} else {
			if tryNum >= maxRetries {
				log.Printf("[%s]: FAILED UPLOAD - %v", f.filePath, err)
				return err
			} else {
				time.Sleep(retryInterval)
				log.Printf("[%s]: %v^RETRYING finish upload...", f.filePath, tryNum)
			}
			tryNum++
		}
	}
	return nil
}

func finishUpload(svc *s3.Client, resp *s3.CreateMultipartUploadOutput, completedParts []types.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
		//ChecksumSHA1:    &checksum,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completedParts},
	}
	return svc.CompleteMultipartUpload(context.TODO(), completeInput)
}

func uploadPartData(svc *s3.Client, resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNumber int, fileName string, buffChan chan int) (types.CompletedPart, error) {
	buffChan <- 0
	defer func() { <-buffChan }()
	log.Printf("[%s]: start upload part#%v", fileName, partNumber)
	tryNum := 0
	partInput := &s3.UploadPartInput{
		Body:          bytes.NewReader(fileBytes),
		Bucket:        resp.Bucket,
		Key:           resp.Key,
		PartNumber:    int32(partNumber),
		UploadId:      resp.UploadId,
		ContentLength: int64(len(fileBytes)),
	}

	for tryNum <= maxRetries {
		uploadResult, err := svc.UploadPart(context.TODO(), partInput)
		if err != nil {
			if tryNum >= maxRetries {
				return types.CompletedPart{}, err
			}
			time.Sleep(retryInterval)
			log.Printf("[%s]: %v^RETRYING upload #%v\n", fileName, tryNum, partNumber)
			tryNum++
		} else {
			log.Printf("[%s]: uploaded #%v\n", fileName, partNumber)
			return types.CompletedPart{
				ETag:       uploadResult.ETag,
				PartNumber: int32(partNumber),
			}, nil
		}
	}
	return types.CompletedPart{}, nil
}

func abortMultipartUpload(svc *s3.Client, resp *s3.CreateMultipartUploadOutput) error {
	log.Println("ABORTING multipart upload for UploadId#" + *resp.UploadId)
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
	}
	_, err := svc.AbortMultipartUpload(context.TODO(), abortInput)
	return err
}

func getFileList(p string) ([]FileInfoT, error) {
	filelist, err := filepath.Glob(p)
	if err != nil {
		return nil, err
	}
	flist := []FileInfoT{}
	for _, filePath := range filelist {
		file, err := os.Open(filePath)
		defer file.Close()
		if err != nil {
			continue
		}
		fInfo, _ := file.Stat()
		if stat, err := file.Stat(); err == nil && !stat.IsDir() {
			flist = append(flist, FileInfoT{filePath, fInfo.Size()})
		}
	}
	return flist, err
}
