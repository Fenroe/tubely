package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

type ffProbeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	fmt.Println(filePath)
	cmd := exec.Command(
		"ffprobe",
		"-v",
		"error",
		"-print_format",
		"json",
		"-show_streams",
		filePath,
	)
	bytesBuffer := bytes.Buffer{}
	cmd.Stdout = &bytesBuffer
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	output := ffProbeOutput{}
	err = json.Unmarshal(bytesBuffer.Bytes(), &output)
	if err != nil {
		fmt.Println("im out 2")
		return "", err
	}
	width := output.Streams[0].Width
	height := output.Streams[0].Height
	if width == 16*height/9 {
		return "16:9", nil
	} else if height == 16*width/9 {
		return "9:16", nil
	}
	return "other", nil
}
