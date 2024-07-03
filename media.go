package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

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
	cmdString := fmt.Sprintf(`yt-dlp -x "%s" --audio-format opus --audio-quality 0 -o audio/%s`, url.String(), id)

	cmd := exec.Command("sh", "-c", cmdString)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error converting video to audio: %s", err)
	}

	if start := url.Query().Get("t"); start != "" {
		seconds, err := strconv.Atoi(start)
		if err != nil {
			return fmt.Errorf("error parsing start time: %s", err)
		}

		parsedTime := time.Unix(0, (time.Duration(seconds) * time.Second).Nanoseconds())
		timeString := strings.Split(parsedTime.String(), " ")[1]

		cmdString := fmt.Sprintf(`ffmpeg -ss %s -i audio/%s.opus -c copy audio/%s_temp.opus -y`, timeString, id, id)

		cmd := exec.Command("sh", "-c", cmdString)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("error cutting audio: %s", err)
		}

		cmdString = fmt.Sprintf(`mv audio/%s_temp.opus audio/%s.opus`, id, id)

		cmd = exec.Command("sh", "-c", cmdString)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("error cutting audio: %s", err)
		}
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

func downloadAttachment(url string) (*Song, error) {
	res, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading attachment: %w", err)
	}
	defer res.Body.Close()
	urlSegments := strings.Split(res.Request.URL.Path, "/")
	fileName := strings.Split(urlSegments[len(urlSegments)-1], "?")[0]
	title := strings.Split(fileName, ".")[0]
	audioPath := fmt.Sprintf("audio/%s", fileName)
	DCAPath := strings.Split(audioPath, ".")[0] + ".dca"

	file, err := os.Create(audioPath)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return nil, fmt.Errorf("error copying file: %w", err)
	}

	cmdString := fmt.Sprintf("ffmpeg -i %s -f s16le -ar 48000 -ac 2 pipe:1 | ./dca > %s", audioPath, DCAPath)
	cmd := exec.Command("sh", "-c", cmdString)
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error converting video to audio: %s", err)
	}

	if err := os.Remove(audioPath); err != nil {
		return nil, fmt.Errorf("error removing audio file: %s", err)
	}

	song := NewSong(title, "", DCAPath)

	return song, nil
}
