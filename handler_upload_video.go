package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	http.MaxBytesReader(w, nil, int64(1<<30))
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve video metadata", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to authenticated user", err)
		return
	}
	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't verify incoming video", err)
		return
	}
	defer file.Close()
	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse filetype", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", errors.New("file must be mp4 format"))
		return
	}
	tempFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temp file", err)
		return
	}
	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write to video temp file", err)
		return
	}
	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get aspect ratio of video", err)
		return
	}
	prefix := aspectRatio
	if aspectRatio == "16:9" {
		prefix = "landscape"
	}
	if aspectRatio == "9:16" {
		prefix = "portrait"
	}
	newFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process video for fast start", err)
		return
	}
	fastStartVideo, err := os.Open(newFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open fast start file", err)
		return
	}
	defer fastStartVideo.Close()
	tempFile.Close()
	os.Remove("tubely-upload.mp4")
	fastStartVideo.Seek(0, io.SeekStart)
	fileNameBits := make([]byte, 32)
	_, err = rand.Read(fileNameBits)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create filename", err)
		return
	}
	fileName := base64.RawURLEncoding.EncodeToString(fileNameBits)
	ext := strings.Split(mediaType, "/")[1]
	fileURL := fmt.Sprintf("%s/%s.%s", prefix, fileName, ext)
	putObjectInput := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileURL,
		Body:        fastStartVideo,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(context.TODO(), &putObjectInput)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload file to storage", err)
		return
	}
	videoURL := fmt.Sprintf("%s/%s", cfg.s3CfDistribution, fileURL)
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video with video url", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
