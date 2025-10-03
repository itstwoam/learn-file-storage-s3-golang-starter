package main

import (
	"fmt"
	"os/exec"
	//"io"
)

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	err := cmd.Run()
	if err != nil {
		fmt.Println("error during ffmpeg operation")
		return "", err
	}
	return outputPath, nil
}

