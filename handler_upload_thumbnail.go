package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)
	// TODO: implement the upload here
	maxMemory := 10 << 20
	r.ParseMultipartForm(int64(maxMemory))
	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't verify thumbnail", err)
		return
	}
	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse filetype", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", errors.New("file must be either png or jpeg format"))
		return
	}
	fileType := strings.Split(fileHeader.Header.Get("Content-Type"), "/")[1]
	fileNameBits := make([]byte, 32)
	_, err = rand.Read(fileNameBits)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create filename", err)
		return
	}
	fileName := base64.RawURLEncoding.EncodeToString(fileNameBits)
	fileURL := fmt.Sprintf("%s.%s", filepath.Join(cfg.assetsRoot, fileName), fileType)
	imageFile, err := os.Create(fileURL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create image file", err)
		return
	}
	_, err = io.Copy(imageFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write to image file", err)
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
	thumbnailURL := fmt.Sprintf("http://localhost:%v/%s", cfg.port, fileURL)
	video.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video with thumbnail url", err)
		return
	}
	respondWithJSON(w, http.StatusOK, video)
}
