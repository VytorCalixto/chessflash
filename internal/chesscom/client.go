package chesscom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vytor/chessflash/internal/logger"
)

type Client struct {
	httpClient *http.Client
	log        *logger.Logger
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		log:        logger.Default().WithPrefix("chesscom"),
	}
}

type archivesResp struct {
	Archives []string `json:"archives"`
}

type MonthlyGame struct {
	URL       string `json:"url"`
	PGN       string `json:"pgn"`
	TimeClass string `json:"time_class"`
	EndTime   int64  `json:"end_time"`
	White     Player `json:"white"`
	Black     Player `json:"black"`
}

type Player struct {
	Username string `json:"username"`
	Result   string `json:"result"`
}

func (c *Client) FetchArchives(ctx context.Context, username string) ([]string, error) {
	log := logger.FromContext(ctx).WithPrefix("chesscom").WithField("username", username)
	url := fmt.Sprintf("https://api.chess.com/pub/player/%s/games/archives", username)

	log.Debug("fetching archives from: %s", url)
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Error("failed to create request: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to fetch archives: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Debug("archives response received in %v, status=%d", time.Since(start), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Error("archives request failed: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("archives status %d: %s", resp.StatusCode, string(body))
	}

	var out archivesResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		log.Error("failed to decode archives response: %v", err)
		return nil, err
	}

	log.Info("fetched %d archives for user %s", len(out.Archives), username)
	return out.Archives, nil
}

func (c *Client) FetchMonthly(ctx context.Context, archiveURL string) ([]MonthlyGame, error) {
	log := logger.FromContext(ctx).WithPrefix("chesscom").WithField("archive_url", archiveURL)

	log.Debug("fetching monthly games")
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		log.Error("failed to create request: %v", err)
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to fetch monthly games: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Debug("monthly response received in %v, status=%d", time.Since(start), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Error("monthly request failed: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("monthly status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Games []MonthlyGame `json:"games"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		log.Error("failed to decode monthly response: %v", err)
		return nil, err
	}

	log.Info("fetched %d games from archive", len(payload.Games))
	return payload.Games, nil
}
