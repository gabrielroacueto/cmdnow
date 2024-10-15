package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

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

var (
	config  Config
	verbose bool
	logger  *log.Logger
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	logger = log.New(os.Stdout, "", log.Ltime)

	configFile, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		fmt.Println("Error parsing config file:", err)
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "scriptgen" {
		fmt.Println("Usage: go run main.go scriptgen <prompt>")
		os.Exit(1)
	}

	userInput := strings.Join(os.Args[2:], " ")
	prompt, err := formatPrompt(userInput)
	if err != nil {
		fmt.Println("Error formatting prompt:", err)
		os.Exit(1)
	}

	command, err := generateBashCommand(prompt)
	if err != nil {
		fmt.Println("Error generating bash command:", err)
		os.Exit(1)
	}

	if command == "" {
		fmt.Println("Warning: Generated command is empty.")
	} else {
		fmt.Println("Generated bash command:")
		fmt.Println(command)
	}
}

func formatPrompt(userInput string) (string, error) {
	tmpl, err := template.New("prompt").Parse(config.Prompts["generate_command"])
	if err != nil {
		return "", fmt.Errorf("error parsing prompt template: %w", err)
	}

	var promptBuffer bytes.Buffer
	err = tmpl.Execute(&promptBuffer, map[string]string{"UserInput": userInput, "Shell": "bash"})
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

	fmt.Println("Sending request to Ollama server...")
	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("error sending request to Ollama server: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Println("Response Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Println("Raw response body:")
	fmt.Println(string(body))

	if len(body) == 0 {
		return "", fmt.Errorf("received empty response from server")
	}

	var ollamaResp OllamaResponse
	err = json.Unmarshal(body, &ollamaResp)
	if err != nil {
		return "", fmt.Errorf("error parsing JSON response: %w", err)
	}

	fmt.Println("Parsed response:")
	fmt.Printf("Model: %s\n", ollamaResp.Model)
	fmt.Println("Response content:")
	fmt.Println(ollamaResp.Response)

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
