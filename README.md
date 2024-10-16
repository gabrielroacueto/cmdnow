# cmdnow

`cmdnow` is a command-line interface (CLI) tool that generates shell commands using a Language Model (LLM). It allows users to describe a task in natural language, and the tool will attempt to generate the appropriate shell command to accomplish that task.

## Features

- Generates shell commands based on user prompts
- Supports multiple shells (bash, zsh, fish)
- Uses a local LLM server (Ollama) for command generation
- Configurable prompts via YAML configuration

## Prerequisites

- Go 1.23.2 or later
- [Ollama](https://ollama.ai/) running locally with the `llama3.2` model (or update the model name in the code)

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/gabrielroacueto/cmdnow.git
   cd cmdnow
   ```

2. Build the project:
   ```
   go build
   ```

3. Ensure Ollama is installed and running:
   - Install Ollama from [https://ollama.ai/](https://ollama.ai/)
   - Start the Ollama server before using `cmdnow`

## Usage

Before using `cmdnow`, make sure the Ollama server is running on your local machine.

```
./cmdnow [options] <prompt>
```

Options:
- `--shell`, `-s`: Specify the shell (bash, zsh, fish). Default is bash.

Example:
```
./cmdnow "list all files in the current directory, including hidden files"
```

## Configuration

The `config.yaml` file contains the prompt template used for generating commands. You can modify this file to customize the prompt sent to the LLM.

## How it Works

1. The user provides a natural language description of a task.
2. The program formats the prompt using the template in `config.yaml`.
3. The formatted prompt is sent to the Ollama server running locally.
4. The LLM generates a response, which is parsed to extract the command.
5. The generated command is displayed to the user.

## Dependencies

- [urfave/cli](https://github.com/urfave/cli): For building the command-line interface
- [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml): For parsing the YAML configuration file
- [Ollama](https://ollama.ai/): For running the local LLM server

## Contributing

Don't contribute yet. I am still working on it.

## License

MIT License

## Disclaimer

This tool generates commands based on AI predictions. Always review and understand any command before executing it on your system.
