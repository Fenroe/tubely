package main

import (
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	splitVideoURL := strings.Split(*video.VideoURL, ",")
	if len(splitVideoURL) < 2 {
		return video, nil
	}
	bucket := splitVideoURL[0]
	key := splitVideoURL[1]
	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*5)
	if err != nil {
		return video, err
	}
	video.VideoURL = &presignedURL
	return video, nil
}
