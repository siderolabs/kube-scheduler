package watttime

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	Username string
	Password string
	BA       string
	token    string
}

type LoginResponse struct {
	Token string `json:"token"`
}

type IndexResponse struct {
	BA        string `json:"ba,omitempty"`
	Freq      string `json:"freq,omitempty"`
	Percent   string `json:"percent,omitempty"`
	PointTime string `json:"point_time,omitempty"`
}

func NewClient(username, password, ba string) *Client {
	return &Client{Username: username, Password: password, BA: ba}
}

func (c *Client) Login() error {
	req, err := http.NewRequest(http.MethodGet, "https://api2.watttime.org/v2/login", nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.Username, c.Password)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make http request: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("failed to read response body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s: %d", string(body), res.StatusCode)
	}

	login := &LoginResponse{}

	err = json.Unmarshal(body, login)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %v", err)
	}

	c.token = login.Token

	return nil
}

func (c *Client) Index() (int, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api2.watttime.org/index", nil)
	if err != nil {
		return -1, err
	}

	q := req.URL.Query()
	q.Add("ba", c.BA)
	q.Add("style", "percent")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return -1, fmt.Errorf("failed to make http request: %w", err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("failed to read response body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("request failed: %s: %d", string(body), res.StatusCode)
	}

	index := IndexResponse{}

	err = json.Unmarshal(body, &index)
	if err != nil {
		return -1, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	n, err := strconv.Atoi(index.Percent)
	if err != nil {
		return -1, fmt.Errorf("failed to convert string to int: %v", err)
	}

	return n, nil
}
