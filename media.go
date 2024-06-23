package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	ytv2 "github.com/kkdai/youtube/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type response struct {
	Items []struct {
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
	} `json:"items"`
}

func IsYouTubeURL(u *url.URL) bool {
	normalizedHost := strings.ToLower(u.Hostname())
	return normalizedHost == "www.youtube.com" || normalizedHost == "youtube.com" || normalizedHost == "youtu.be"
}

func GetSongID(u url.URL) ([]string, error) {
	var ids []string

	if strings.Contains(u.Path, "/playlist") { // playlist yt link
		ids, err = parsePlaylist(u)
		if err != nil {
			return nil, err
		}
	} else if strings.Contains(u.Path, "/watch") { // normal yt link
		ids = append(ids, u.Query().Get("v"))
	} else if strings.Contains(u.Path, "/shorts/") { // shorts yt link
		ids = append(ids, strings.Split(u.Path, "/shorts/")[1])
	} else { // shorten yt link
		ids = append(ids, u.Path)
	}

	return ids, nil
}

func GetSongTitle(id string) (string, error) {
	service, err := youtube.NewService(
		context.Background(),
		option.WithAPIKey(YT),
	)
	if err != nil {
		return "", fmt.Errorf("error creating yt service: %w", err)
	}
	call := service.Videos.List([]string{"snippet"})
	call = call.Id(id)
	resp, err := call.Do()
	if err != nil {
		return "", fmt.Errorf("error getting playlist data: %w", err)
	}
	var title string
	for _, video := range resp.Items {
		title = video.Snippet.Title
	}

	return title, nil
}

func GetSongTitleOld(id string) (string, error) {
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?id=%s&key=%s&fields=items(snippet(title))&part=snippet",
		id,
		YT,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	var response response

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	title := response.Items[0].Snippet.Title

	return title, nil
}

func DownloadSong(id string) (string, error) {
	err := downloadVideo(id)
	if err != nil {
		return "", err
	}

	audioPath, err := convertToAudio(id)
	if err != nil {
		return "", err
	}

	return audioPath, nil
}

func downloadVideo(id string) error {
	client := ytv2.Client{}

	video, err := client.GetVideo(id)
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}

	formats := video.Formats.WithAudioChannels() // only get videos with audio

	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}
	defer stream.Close()

	file, err := os.Create(fmt.Sprintf("video/%s.mp4", id))
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		os.Remove(fmt.Sprintf("video/%s.mp4", id))
		return fmt.Errorf("error downloading video: %s", err)
	}

	return nil
}

func convertToAudio(id string) (string, error) {
	audioPath := fmt.Sprintf("audio/%s.dca", id)
	videoPath := fmt.Sprintf("video/%s.mp4", id)

	cmdString := fmt.Sprintf("ffmpeg -i %s -f s16le -ar 48000 -ac 2 pipe:1 | ./dca > %s", videoPath, audioPath)
	cmd := exec.Command("sh", "-c", cmdString)
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("error converting video to audio: %s", err)
	}

	return audioPath, nil
}

func parsePlaylist(u url.URL) ([]string, error) {
	service, err := youtube.NewService(
		context.Background(),
		option.WithAPIKey(YT),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating yt service: %w", err)
	}

	call := service.PlaylistItems.List([]string{"snippet"})
	call = call.MaxResults(50)
	call = call.PlaylistId(u.Query().Get("list"))
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("error getting playlist data: %w", err)
	}

	videos := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		videos = append(videos, item.Snippet.ResourceId.VideoId)
	}

	return videos, nil
}
