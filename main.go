package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
}

type Config struct {
	Prompts map[string]string `yaml:"prompts"`
}

var config Config

func init() {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Error reading config file: ", err)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Error parsing config file: ", err)
	}
}

func main() {
	app := &cli.App{
		Name:  "scriptgen",
		Usage: "Generate shell commands using LLM",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "shell",
				Aliases: []string{"s"},
				Value:   "bash",
				Usage:   "Specify the shell (bash, zsh, fish)",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Error: Please provide a prompt for command generation", 1)
			}

			shell := c.String("shell")
			userInput := strings.Join(c.Args().Slice(), " ")

			return generateCommand(userInput, shell)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func generateCommand(userInput string, shell string) error {
	prompt, err := formatPrompt(userInput, shell)
	if err != nil {
		return fmt.Errorf("Error formating prompt: %w", err)
	}

	response, err := generateBashCommand(prompt)
	if err != nil {
		return fmt.Errorf("Error generating command: %w", err)
	}

	fmt.Println(response)

	return nil
}

func formatPrompt(userInput string, shell string) (string, error) {
	tmpl, err := template.New("prompt").Parse(config.Prompts["generate_command"])
	if err != nil {
		return "", fmt.Errorf("error parsing prompt template: %w", err)
	}

	var promptBuffer bytes.Buffer
	err = tmpl.Execute(&promptBuffer, map[string]string{"UserInput": userInput, "Shell": shell})
	if err != nil {
		return "", fmt.Errorf("error executing prompt template: %w", err)
	}

	return promptBuffer.String(), nil
}

func generateBashCommand(prompt string) (string, error) {
	payload := OllamaRequest{
		Model:  "llama3.2", // Make sure this matches the model you're using
		Prompt: prompt,
		Stream: false,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error creating JSON payload: %w", err)
	}

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error sending request to Ollama server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if len(body) == 0 {
		return "", fmt.Errorf("received empty response from server")
	}

	var ollamaResp OllamaResponse
	err = json.Unmarshal(body, &ollamaResp)
	if err != nil {
		return "", fmt.Errorf("error parsing JSON response: %w", err)
	}

	if ollamaResp.Response == "" {
		return "", fmt.Errorf("LLM returned an empty response")
	}

	return parseCommandFromResponse(ollamaResp.Response)
}

func parseCommandFromResponse(response string) (string, error) {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "COMMAND:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "COMMAND:")), nil
		}
	}
	return "", fmt.Errorf("no command found in response")
}
