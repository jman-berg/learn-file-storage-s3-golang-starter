package main

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const maxMemory = 10 << 20

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

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing thumbnail for video", err)
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find file in request", err)
	}

	mediaType := fileHeader.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusInternalServerError, "No media type found in header", err)
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading image data", err)
	}

	base64Video := base64.StdEncoding.EncodeToString(fileBytes)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			respondWithError(w, http.StatusNotFound, "video does not exist", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Error fetching video", err)
	}

	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "This is not your video, buddy", err)
	}

	NewThumbnailURL := fmt.Sprintf("data:%s;base64,%s", mediaType, base64Video)
	video.ThumbnailURL = &NewThumbnailURL

	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video with new thumbnail url", err)
	}

	respondWithJSON(w, http.StatusOK, video)
}
