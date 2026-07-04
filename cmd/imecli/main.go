package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
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

type wordbookResponse struct {
	UserScores map[string]int `json:"userScores"`
	Count      int            `json:"count"`
	UpdatedAt  string         `json:"updatedAt"`
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
	case "wordbook":
		err = wordbook(client, os.Args[2:])
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
  shurufa-imecli wordbook list
  shurufa-imecli wordbook export
  shurufa-imecli wordbook import user-wordbook.json [--replace]
  shurufa-imecli wordbook delete "nihao|你好"
  shurufa-imecli wordbook clear
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

func wordbook(client *http.Client, args []string) error {
	action := "list"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
		args = args[1:]
	}
	switch action {
	case "list":
		var response wordbookResponse
		if err := getJSON(client, "/wordbook", &response); err != nil {
			return err
		}
		printWordbook(response.UserScores)
		return nil
	case "export":
		var response wordbookResponse
		if err := getJSON(client, "/wordbook", &response); err != nil {
			return err
		}
		data, err := json.MarshalIndent(map[string]any{"userScores": response.UserScores}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	case "import":
		if len(args) == 0 {
			return fmt.Errorf("missing import file")
		}
		scores, err := readWordbookFile(args[0])
		if err != nil {
			return err
		}
		payload := map[string]any{"userScores": scores, "merge": true}
		if len(args) > 1 && args[1] == "--replace" {
			payload["merge"] = false
		}
		var response wordbookResponse
		if err := putJSON(client, "/wordbook", payload, &response); err != nil {
			return err
		}
		fmt.Printf("imported=%d\n", response.Count)
		return nil
	case "delete":
		if len(args) == 0 {
			return fmt.Errorf("missing wordbook key")
		}
		var response wordbookResponse
		if err := deleteJSON(client, "/wordbook?key="+urlQueryEscape(args[0]), &response); err != nil {
			return err
		}
		fmt.Printf("remaining=%d\n", response.Count)
		return nil
	case "clear":
		var response wordbookResponse
		if err := deleteJSON(client, "/wordbook", &response); err != nil {
			return err
		}
		fmt.Printf("remaining=%d\n", response.Count)
		return nil
	default:
		return fmt.Errorf("unknown wordbook action %q", action)
	}
}

func printWordbook(scores map[string]int) {
	keys := make([]string, 0, len(scores))
	for key := range scores {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if scores[keys[i]] == scores[keys[j]] {
			return keys[i] < keys[j]
		}
		return scores[keys[i]] > scores[keys[j]]
	})
	for _, key := range keys {
		fmt.Printf("%s\t%d\n", key, scores[key])
	}
}

func readWordbookFile(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var wrapped struct {
		UserScores map[string]int `json:"userScores"`
		Scores     map[string]int `json:"scores"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && (wrapped.UserScores != nil || wrapped.Scores != nil) {
		if wrapped.UserScores != nil {
			return wrapped.UserScores, nil
		}
		return wrapped.Scores, nil
	}
	var scores map[string]int
	if err := json.Unmarshal(data, &scores); err != nil {
		return nil, err
	}
	return scores, nil
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

func putJSON(client *http.Client, path string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := request(client, http.MethodPut, path, data)
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
	return request(client, http.MethodPost, path, body)
}

func deleteJSON(client *http.Client, path string, out any) error {
	body, err := request(client, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func request(client *http.Client, method string, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, apiBase+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return readResponse(resp)
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
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
