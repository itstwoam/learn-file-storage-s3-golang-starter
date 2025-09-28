package main

import (
	"fmt"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"encoding/base64"
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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	
	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "missing Content-Type", nil)
		return
	}

	imageData, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("failed to read thumbnail image")
		respondWithError(w, http.StatusInternalServerError, "", nil)
		return
	}
	imageDataString := base64.StdEncoding.EncodeToString(imageData)
	url := makeDataURL(imageDataString, mediaType, "base64")

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		fmt.Println("failed to get video metadata")
		respondWithError(w, http.StatusInternalServerError, "", nil)
		return
	}

	if metadata.CreateVideoParams.UserID != userID {
		fmt.Println("userID does not match video owner ID during thumbnail upload")
		respondWithError(w, http.StatusUnauthorized, "Uploading user does not match video's user", nil)
		return
	}

	metadata.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		fmt.Println("error while updating video metadata")
		respondWithError(w, http.StatusInternalServerError, "", nil)
	}
	respondWithJSON(w, http.StatusOK, metadata)
}

func makeDataURL(data, mediaType, coding string) string {
	final := "data:"
	if mediaType != "" {
		final = final + mediaType
	}
	if coding != "" {
		final = final + ";"+coding
	}
	final = final + ","+data
	return final
}
