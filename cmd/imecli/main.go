package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/neko233-com/shurufa233/core/engine"
)

func main() {
	e := engine.New(engine.DefaultConfig())
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("shurufa233 CLI. Type pinyin, /select N, /back, /clear, /quit.")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "/quit":
			return
		case line == "/back":
			printState(e.Backspace())
		case line == "/clear":
			printState(e.Clear())
		case strings.HasPrefix(line, "/select "):
			indexText := strings.TrimSpace(strings.TrimPrefix(line, "/select "))
			index, err := strconv.Atoi(indexText)
			if err != nil {
				fmt.Println("invalid index")
				continue
			}
			state, err := e.Select(index - 1)
			if err != nil {
				fmt.Println(err)
				continue
			}
			printState(state)
		default:
			printState(e.Preview(line))
		}
	}
}

func printState(state engine.State) {
	if state.Committed != "" {
		fmt.Println("commit:", state.Committed)
		return
	}
	fmt.Println("buffer:", state.Buffer)
	for i, candidate := range state.Candidates {
		fmt.Printf("%d. %s [%s] score=%d\n", i+1, candidate.Text, candidate.Reading, candidate.Weight+candidate.UserScore)
	}
}
