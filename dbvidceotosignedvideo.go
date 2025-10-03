package main

import (
	"fmt"
	"strings"
	"errors"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"time"
	//"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil || *video.VideoURL == "" {
		return video, nil
	}
	split := strings.Split(*video.VideoURL, ",")
	if len(split) < 2 || len(split[0]) < 1 || len(split[1]) < 1 {
		return database.Video{}, errors.New("failed to extract bucket or key")
	}
	expiryTime, err := time.ParseDuration("10m")
	if err != nil {
		return database.Video{}, errors.New("error when parsing time duration")
	}
	presigned, err := generatePresignedURL(cfg.s3Client, split[0], split[1], expiryTime)
	if err != nil {
		return database.Video{}, err
	}
	video.VideoURL = &presigned
	return video, nil
}
