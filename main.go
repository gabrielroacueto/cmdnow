package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

func main() {
	// Check if the command is "scriptgen"
	if len(os.Args) < 2 || os.Args[1] != "scriptgen" {
		fmt.Println("Usage: go run main.go scriptgen <prompt>")
		os.Exit(1)
	}

	// Get the prompt from command line arguments
	prompt := strings.Join(os.Args[2:], " ")

	// Prepare the request payload
	payload := OllamaRequest{
		Model:  "llama3.2", // You can change this to the model you're using
		Prompt: prompt,
		Stream: false, // Set to false for non-streaming response
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error creating JSON payload:", err)
		os.Exit(1)
	}

	// Send POST request to Ollama server
	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error sending request to Ollama server:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Print the response status code and headers for debugging
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Println("Response Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", key, value)
		}
	}

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		os.Exit(1)
	}

	// Print the raw response body for debugging
	fmt.Println("Raw response body:")
	fmt.Println(string(body))

	// Parse the JSON response
	var ollamaResp OllamaResponse
	err = json.Unmarshal(body, &ollamaResp)
	if err != nil {
		fmt.Println("Error parsing JSON response:", err)
		os.Exit(1)
	}

	// Print the parsed response
	fmt.Println("\nParsed response:")
	fmt.Printf("Model: %s\n", ollamaResp.Model)
	fmt.Println("Generated response:")
	fmt.Println(ollamaResp.Response)

	if ollamaResp.Response == "" {
		fmt.Println("Warning: The generated response is empty.")
	}
}

