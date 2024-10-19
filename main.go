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

type GenerateCommandOptions struct {
	Shell         string
	ShouldExplain bool
}

func main() {
	appConfig, err := loadConfig()

	if err != nil {
		os.Exit(1)
	}

	app := &cli.App{
		Name:  "cmdnow",
		Usage: "Generate shell commands using LLM",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "shell",
				Aliases: []string{"s"},
				Value:   "bash",
				Usage:   "Specify the shell (bash, zsh, fish)",
			},
			&cli.BoolFlag{
				Name:    "explain",
				Aliases: []string{"e"},
				Value:   false,
				Usage:   "Explain the command generated.",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				return cli.Exit("Error: Please provide a prompt for command generation", 1)
			}

			shell := c.String("shell")
			shouldExplain := c.Bool("explain")
			userInput := strings.Join(c.Args().Slice(), " ")

			command_options := GenerateCommandOptions{
				Shell:         shell,
				ShouldExplain: shouldExplain,
			}

			return generateCommand(userInput, appConfig, command_options)
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func loadConfig() (Config, error) {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Error reading config file: ", err)
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Error parsing config file: ", err)
		return Config{}, err
	}

	return config, nil
}

func generateCommand(userInput string, appConfig Config, commandOptions GenerateCommandOptions) error {
	prompt, err := generateCommandPrompt(appConfig, userInput, commandOptions.Shell)
	if err != nil {
		return fmt.Errorf("Error formating prompt: %w", err)
	}

	response, err := generateBashCommand(prompt)
	if err != nil {
		return fmt.Errorf("Error generating command: %w", err)
	}

	fmt.Println(response)

	if commandOptions.ShouldExplain == true {
		explanationPrompt, err := generateExplainCommandPrompt(appConfig, response)

		if err != nil {
			return fmt.Errorf("Error generaing explanation prompt: %w", err)
		}

		explanationResponse, err := generateCommandExplanation(explanationPrompt)

		if err != nil {
			return fmt.Errorf("Error generating explanation from LLM: %w", err)
		}

		explanationContent, err := parseExplanationFromResponse(explanationResponse)

		if err != nil {
			return fmt.Errorf("Error parsing explanation content from response: %w", err)
		}

		fmt.Println(explanationContent)

	}

	return nil
}

func generateCommandPrompt(config Config, userInput string, shell string) (string, error) {
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

func generateExplainCommandPrompt(config Config, command string) (string, error) {
	tmpl, err := template.New("prompt").Parse(config.Prompts["explain_command"])
	if err != nil {
		return "", fmt.Errorf("error parsing prompt template: %w", err)
	}

	var promptBuffer bytes.Buffer
	err = tmpl.Execute(&promptBuffer, map[string]string{"Command": command})
	if err != nil {
		return "", fmt.Errorf("error executing prompt template: %w", err)
	}

	return promptBuffer.String(), nil
}

func generateBashCommand(prompt string) (string, error) {
	resp, err := ollamaGenerate(prompt)
	if err != nil {
		return "", fmt.Errorf("error generating prompt with ollama: %w", err)
	}

	return parseCommandFromResponse(resp)
}

func generateCommandExplanation(prompt string) (string, error) {
	resp, err := ollamaGenerate(prompt)
	if err != nil {
		return "", fmt.Errorf("error generating prompt with ollama: %w", err)
	}

	return resp, nil
}

func ollamaGenerate(prompt string) (string, error) {
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

	return ollamaResp.Response, nil
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

func parseExplanationFromResponse(response string) (string, error) {
	return parseXmlContent(response, "explanation")
}

func parseXmlContent(text string, tag string) (string, error) {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"

	startIndex := strings.Index(text, openTag)
	if startIndex == -1 {
		return "", fmt.Errorf("opening tag '%s' not found", tag)
	}

	// Move start index to after the opening tag
	startIndex += len(openTag)

	endIndex := strings.Index(text[startIndex:], closeTag)
	if endIndex == -1 {
		return "", fmt.Errorf("closing tag '%s' not found", tag)
	}

	// endIndex is relative to startIndex, so we don't need to add startIndex here
	return text[startIndex : startIndex+endIndex], nil
}
