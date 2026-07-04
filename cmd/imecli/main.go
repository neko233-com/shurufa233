package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const apiBase = "http://127.0.0.1:23333"

type candidate struct {
	Text      string `json:"text"`
	Reading   string `json:"reading"`
	Kind      string `json:"kind,omitempty"`
	Source    string `json:"source,omitempty"`
	Weight    int    `json:"weight"`
	UserScore int    `json:"userScore"`
}

type engineState struct {
	Buffer     string      `json:"buffer"`
	Mode       string      `json:"mode"`
	Candidates []candidate `json:"candidates"`
	Committed  string      `json:"committed,omitempty"`
}

type updateCheck struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ManifestURL     string `json:"manifestUrl"`
}

type agentResponse struct {
	Input      string           `json:"input"`
	Context    string           `json:"context,omitempty"`
	Candidates []string         `json:"candidates"`
	Items      []agentCandidate `json:"items"`
	Actions    []string         `json:"actions"`
}

type agentCandidate struct {
	Text    string `json:"text"`
	Intent  string `json:"intent"`
	Action  string `json:"action"`
	Source  string `json:"source"`
	Context string `json:"context,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	var err error
	switch os.Args[1] {
	case "status":
		err = status(client)
	case "preview":
		err = preview(client, strings.Join(os.Args[2:], " "))
	case "update-check":
		err = updateCheckCmd(client)
	case "update-apply":
		err = updateApply(client)
	case "mode":
		err = mode(client, os.Args[2:])
	case "agent":
		err = agent(client, os.Args[2:])
	case "repl":
		err = repl(client)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`shurufa233 CLI

Usage:
  shurufa-imecli status
  shurufa-imecli preview nihao
  shurufa-imecli mode [zh|en|toggle]
  shurufa-imecli update-check
  shurufa-imecli update-apply
  shurufa-imecli agent "/rewrite hello"
  shurufa-imecli agent --context "selected text" "/rewrite"
  shurufa-imecli repl`)
}

func status(client *http.Client) error {
	body, err := get(client, "/health")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func preview(client *http.Client, input string) error {
	if input == "" {
		return fmt.Errorf("missing preview input")
	}
	var state engineState
	if err := postJSON(client, "/engine/preview", map[string]string{"input": input}, &state); err != nil {
		return err
	}
	fmt.Printf("buffer: %s\n", state.Buffer)
	for i, item := range state.Candidates {
		meta := ""
		if item.Kind != "" {
			meta = " kind=" + item.Kind
			if item.Source != "" {
				meta += " source=" + item.Source
			}
		}
		fmt.Printf("%d. %s [%s] score=%d%s\n", i+1, item.Text, item.Reading, item.Weight+item.UserScore, meta)
	}
	return nil
}

func updateCheckCmd(client *http.Client) error {
	var check updateCheck
	if err := getJSON(client, "/updates/check", &check); err != nil {
		return err
	}
	fmt.Printf("current=%s latest=%s available=%v\n", check.CurrentVersion, check.LatestVersion, check.UpdateAvailable)
	fmt.Println(check.ManifestURL)
	return nil
}

func updateApply(client *http.Client) error {
	body, err := post(client, "/updates/apply", nil)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func mode(client *http.Client, args []string) error {
	if len(args) == 0 {
		var state engineState
		if err := getJSON(client, "/ime/mode", &state); err != nil {
			return err
		}
		fmt.Println(state.Mode)
		return nil
	}
	next := strings.TrimSpace(strings.ToLower(strings.Join(args, " ")))
	payload := map[string]any{"mode": next}
	if next == "toggle" {
		payload = map[string]any{"toggle": true}
	}
	var state engineState
	if err := postJSON(client, "/ime/mode", payload, &state); err != nil {
		return err
	}
	fmt.Println(state.Mode)
	return nil
}

func agent(client *http.Client, args []string) error {
	input, context := parseAgentArgs(args)
	if input == "" {
		return fmt.Errorf("missing agent input")
	}
	var response agentResponse
	payload := map[string]string{"input": input}
	if context != "" {
		payload["context"] = context
	}
	if err := postJSON(client, "/agent/compose", payload, &response); err != nil {
		return err
	}
	if len(response.Items) > 0 {
		for i, item := range response.Items {
			fmt.Printf("%d. %s [%s/%s]\n", i+1, item.Text, item.Intent, item.Action)
		}
		return nil
	}
	for i, item := range response.Candidates {
		fmt.Printf("%d. %s\n", i+1, item)
	}
	return nil
}

func parseAgentArgs(args []string) (string, string) {
	var context string
	var input []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--context", "-c":
			if i+1 < len(args) {
				context = args[i+1]
				i++
			}
		default:
			input = append(input, args[i])
		}
	}
	return strings.TrimSpace(strings.Join(input, " ")), strings.TrimSpace(context)
}

func repl(client *http.Client) error {
	fmt.Println("shurufa233 CLI REPL. Type pinyin, /agent text, /quit.")
	var line string
	for {
		fmt.Print("> ")
		if _, err := fmt.Scanln(&line); err != nil {
			return nil
		}
		if line == "/quit" {
			return nil
		}
		if strings.HasPrefix(line, "/agent") {
			args := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "/agent")))
			if err := agent(client, args); err != nil {
				fmt.Println(err)
			}
			continue
		}
		if err := preview(client, line); err != nil {
			fmt.Println(err)
		}
	}
}

func getJSON(client *http.Client, path string, out any) error {
	body, err := get(client, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func postJSON(client *http.Client, path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := post(client, path, data)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func get(client *http.Client, path string) ([]byte, error) {
	resp, err := client.Get(apiBase + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readResponse(resp)
}

func post(client *http.Client, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	resp, err := client.Post(apiBase+path, "application/json", reader)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readResponse(resp)
}

func readResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned HTTP %d: %s", resp.Request.URL.Path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
