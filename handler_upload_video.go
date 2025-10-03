package main

import (
	"fmt"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	//"encoding/base64"
	//"strings"
	"os"
	//"path"
	"mime"
	"crypto/rand"
	"math/big"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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
		respondWithError(w, http.StatusUnauthorized, "user validation error", err)
		return
	}

	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		fmt.Println("failed to get video metadata")
		respondWithError(w, http.StatusInternalServerError, "", nil)
		return
	}

	if videoMetaData.CreateVideoParams.UserID != userID {
		fmt.Println("userID does not match video owner ID during video upload")
		respondWithError(w, http.StatusUnauthorized, "Uploading user does not match video's user", nil)
		return
	}

	const maxMemory = 1024 * 1024 * 1024

	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "missing/malformed/not-allowed Content-Type", nil)
		return
	}

	tFile, err := os.CreateTemp("", "tubely-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "problem creating temporary file", err)
		return
	}

	_, err = io.Copy(tFile, file)
	if err != nil {
		fmt.Println("error when copying file to temp")
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}
	tFile.Close()
	pFileName, err := processVideoForFastStart(tFile.Name())
	if err != nil {
		fmt.Println("error preprocessing temp file")
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}
	pFile, err := os.Open(pFileName)
	if err != nil {
		fmt.Println("error opening processing file")
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}


	defer pFile.Close()
	//tFile.Seek(0, io.SeekStart)
	defer os.Remove(pFile.Name())	
	defer os.Remove(tFile.Name())
	aspectRatio, err := getVideoAspectRatio(pFile.Name())
	if err != nil {
		aspectRatio = "other"
	}
	aspectString := "other"
	if aspectRatio == "16:9" {
		aspectString = "landscape"
	}else if aspectRatio == "9:16" {
		aspectString = "portrait"
	}
	
	fileKey, err := randomString(32, true)
	if err != nil {
		fmt.Println("error creating key name")
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}
	fileKey = aspectString + "/" + fileKey + ".mp4"

	input := &s3.PutObjectInput{
		Bucket: aws.String(cfg.s3Bucket),
		Key: aws.String(fileKey),
		Body: pFile,
		ContentType: aws.String(mediaType),
	}
	
	fmt.Println("Uploading video ", videoID, "by user", userID)
	_, err = cfg.s3Client.PutObject(r.Context(), input)
	if err != nil {
		fmt.Println("error when uploading file to s3")
		respondWithError(w, http.StatusInternalServerError, "", err)
		return
	}

	// videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileKey)
	videoURL := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileKey)
	videoMetaData.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(videoMetaData)
	if err != nil {
		fmt.Println("error while updating video metadata")
		respondWithError(w, http.StatusInternalServerError, "error while updating video metadata", err)
		return
	}
	videoMetaData, err = cfg.dbVideoToSignedVideo(videoMetaData)
	if err != nil {
		fmt.Println("error when converting video url")
		respondWithError(w, http.StatusInternalServerError, "error when converting video url", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoMetaData)
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const hexcharset = "abcdef0123456789" 

func randomString(length int, useHex bool) (string, error) {
    result := make([]byte, length)
    for i := range result {
        var (
            n   *big.Int
            err error
        )
        if !useHex {
            n, err = rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        } else {
            n, err = rand.Int(rand.Reader, big.NewInt(int64(len(hexcharset))))
        }
        if err != nil {
            return "", err
        }
        if !useHex {
            result[i] = charset[n.Int64()]
        } else {
            result[i] = hexcharset[n.Int64()]
        }
    }
    return string(result), nil
}

