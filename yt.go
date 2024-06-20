package musicbot

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/kkdai/youtube/v2"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type Song struct {
	Title     string
	ID        string
	URL       url.URL
	VideoPath string
	AudioPath string
}

type Item struct {
	Snippet Snippet `json:"snippet"`
}

type Snippet struct {
	Title string `json:"title"`
}

type Response struct {
	Items []Item `json:"items"`
}

func IsYouTubeURL(u *url.URL) bool {
	normalizedHost := strings.ToLower(u.Hostname())
	return normalizedHost == "www.youtube.com" || normalizedHost == "youtube.com" || normalizedHost == "youtu.be"
}

func GetSongID(u url.URL) string {
	var id string
	if u.Path == "watch" {
		id = u.Query().Get("v")
	} else {
		id = u.Path
	}
	return id
}

func GetSongTitle(u url.URL) (string, error) {
	id := GetSongID(u)
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?id=%s&key=%s&fields=items(snippet(title))&part=snippet",
		id,
		YT,
	)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("getSongTitle: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getSongTitle: %w", err)
	}

	var response Response

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("getSongTitle: failed to read response body: %w", err)
	}

	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		log.Fatalf("getSongTitle: Error unmarshalling JSON: %v", err)
	}

	title := response.Items[0].Snippet.Title

	return title, nil
}

func (s *Song) Download() error {
	if err := s.downloadVideo(); err != nil {
		return err
	}

	if err := s.convertToAudio(); err != nil {
		return err
	}

	return nil
}

func (s *Song) downloadVideo() error {
	client := youtube.Client{}

	video, err := client.GetVideo(s.ID)
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}

	formats := video.Formats.WithAudioChannels() // only get videos with audio

	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}
	defer stream.Close()

	file, err := os.Create(fmt.Sprintf("video/%s.mp4", s.ID))
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		return fmt.Errorf("error downloading video: %s", err)
	}

	s.VideoPath = fmt.Sprintf("video/%s.mp4", s.ID)

	return nil
}

func (s *Song) convertToAudio() error {
	err := ffmpeg_go.
		Input(s.VideoPath).
		Output(fmt.Sprintf("audio/%s.mp3", s.ID)).
		Run()
	if err != nil {
		return fmt.Errorf("error converting video to audio: %s", err)
	}

	s.AudioPath = fmt.Sprintf("audio/%s.mp3", s.ID)

	return nil
}
