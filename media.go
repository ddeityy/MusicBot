package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Season int

const (
	YouTube Season = iota
	VK
	Winter
	Spring
)

type song struct {
	url url.URL
	t   Season
}

func IsYouTubeURL(u *url.URL) bool {
	normalizedHost := strings.ToLower(u.Hostname())
	return normalizedHost == "www.youtube.com" || normalizedHost == "youtube.com" || normalizedHost == "youtu.be"
}

func GetSongID(u url.URL) ([]string, error) {
	var err error
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

func DownloadSong(url url.URL, id string) (string, error) {
	if err := downloadAudio(url, id); err != nil {
		return "", fmt.Errorf("error downloading audio: %w", err)
	}

	if err := convertToDCA(id); err != nil {
		return "", fmt.Errorf("error converting to dca: %w", err)
	}

	return fmt.Sprintf("audio/%s.dca", id), nil
}

func convertToDCA(id string) error {
	audioPath := fmt.Sprintf("audio/%s.opus", id)
	DCAPath := fmt.Sprintf("audio/%s.dca", id)

	cmdString := fmt.Sprintf("ffmpeg -i %s -f s16le -ar 48000 -ac 2 pipe:1 | ./dca > %s", audioPath, DCAPath)
	cmd := exec.Command("sh", "-c", cmdString)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error converting video to audio: %s", err)
	}

	if err := os.Remove(audioPath); err != nil {
		return fmt.Errorf("error removing audio file: %s", err)
	}

	return nil
}

func downloadAudio(url url.URL, id string) error {
	cmdString := fmt.Sprintf(`yt-dlp -x "%s" -o audio/%s`, url.String(), id)

	cmd := exec.Command("sh", "-c", cmdString)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error converting video to audio: %s", err)
	}

	return nil
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
