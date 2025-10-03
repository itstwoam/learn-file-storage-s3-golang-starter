package main

import (
	"fmt"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"encoding/base64"
	"strings"
	"os"
	"path"
	"mime"
	"crypto/rand"
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

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "missing/malformed/not-allowed Content-Type", nil)
		return
	}

	extSlice := strings.Split(mediaType, "/")
	ext := extSlice[1]
	randBytes := make([]byte, 32)
	rand.Read(randBytes)
	encString := base64.RawURLEncoding.EncodeToString(randBytes)
	thumbnailName := path.Join(cfg.assetsRoot, encString + "." + ext)
	newFile, err := os.Create(thumbnailName)
	if err != nil {
		fmt.Println("error when creating new thumbnail file in assets")
		respondWithError(w, http.StatusInternalServerError, "", nil)
		return
	}
	defer newFile.Close()
	
	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println("error when copying datat to new file")
		respondWithError(w, http.StatusInternalServerError, "", nil)
		return
	}

	url := fmt.Sprintf("http://%s:%s/assets/%s.%s", cfg.hname, cfg.port, encString, ext)

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
