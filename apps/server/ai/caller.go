package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type responseRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type responsesAPIResponse struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	Error  *apiError    `json:"error"`
	Output []outputItem `json:"output"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

type outputItem struct {
	Type    string        `json:"type"`
	Role    string        `json:"role"`
	Content []contentItem `json:"content"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func extractOutputText(resp responsesAPIResponse) string {
	var b strings.Builder
	for _, item := range resp.Output {
		if item.Type != "message" || item.Role != "assistant" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				b.WriteString(c.Text)
			}
		}
	}
	return b.String()
}

func getCallParameters() (string, string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Missing OPENAI_API_KEY environment variable")
		return "", "", errors.New("No API key")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-5-nano"
	}

	return apiKey, model, nil
}

func createHTTPRequest(model string, prompt string, apiKey string, ctx context.Context) (*http.Request, error) {
	reqBody := responseRequest{
		Model: model,
		Input: prompt,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintln(os.Stderr, "JSON marshal error:", err)
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/responses",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewRequest error:", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	return httpReq, nil
}

func readResponse(httpResp *http.Response) (string, error) {

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ReadAll error:", err)
		return "", err
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "HTTP %d\n%s\n", httpResp.StatusCode, string(respBytes))
		return "", errors.New("Bad HTTP status code")
	}

	var parsed responsesAPIResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		fmt.Fprintln(os.Stderr, "JSON unmarshal error:", err)
		fmt.Fprintln(os.Stderr, "Raw response:", string(respBytes))
		return "", err
	}

	if parsed.Error != nil {
		fmt.Fprintf(os.Stderr, "API error: %s (type=%s)\n", parsed.Error.Message, parsed.Error.Type)
		return "", err
	}

	text := extractOutputText(parsed)
	if text == "" {
		fmt.Println(string(respBytes))
		return "", errors.New("Empty response")
	}

	return text, nil
}

func callChatAPI(prompt string) (string, error) {

	apiKey, model, err := getCallParameters()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	httpReq, err := createHTTPRequest(model, prompt, apiKey, ctx)

	if err != nil {
		return "", err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Request error:", err)
		return "", err
	}
	defer httpResp.Body.Close()

	text, err := readResponse(httpResp)
	if err != nil {
		return "", err
	}

	return text, nil

}
