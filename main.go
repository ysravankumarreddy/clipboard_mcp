package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ysravankumarreddy/mcp-protocol"
	"golang.design/x/clipboard"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "[Clipboard-MCP]", log.Ltime)
}

func main() {
	logger.Println("Starting Clipboard-MCP Server...")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		var req mcp.Request
		if err := json.Unmarshal(line, &req); err != nil {
			logger.Printf("Failed to unmarshal request: %v", err)
			continue
		}

		logger.Printf("Recieved method: %v", req.Method)
		dispatchRequest(req)
	}
	if err := scanner.Err(); err != nil {
		logger.Fatalf("Scanner error: %v", err)
	}
}

func handleToolList(id interface{}) {
	result := mcp.ToolListResult{
		Tools: []mcp.Tool{
			{
				Name:        "read_clipboard_text",
				Description: "Reads the current text contents of the user's Windows clipboard. Use this when the user asks you to look at copied text, code, or error messages.",
				InputSchema: mcp.InputSchema{
					Type:       "object",
					Properties: make(map[string]mcp.PropertySchema),
					Required:   []string{},
				},
			},
			{
				Name:        "write_to_clipboard_text",
				Description: "Write the text to user's Windows clipboard, Use this when the user ask you to copy respose, text, code, or error messages.",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"text_to_copy": {
							Type:        "string",
							Description: "The exact text or code snippet to place into the clipboard",
						},
					},
					Required: []string{"text_to_copy"},
				},
			},
			{
				Name:        "read_clipboard_image",
				Description: "Reads the image currently copied in user's Windows clipboard. returned a base64",
				InputSchema: mcp.InputSchema{
					Type:       "object",
					Properties: make(map[string]mcp.PropertySchema),
					Required:   []string{},
				},
			},
			{
				Name:        "write_to_clipboard_image",
				Description: "Writes a base64 enacoded png image to Windows clipboard.",
				InputSchema: mcp.InputSchema{
					Type: "object",
					Properties: map[string]mcp.PropertySchema{
						"base64_png": {
							Type:        "string",
							Description: "base64 encoded png image",
						},
					},
					Required: []string{"base64_png"},
				},
			},
		},
	}
	writeResponse(id, result)
}

func dispatchRequest(req mcp.Request) {
	switch req.Method {
	case "initialize":
		logger.Printf("initialized")
		handleInitialize(req.ID)
	case "tools/list":
		logger.Printf("List of tools")
		handleToolList(req.ID)
	case "tools/call":
		logger.Printf("execute call")
		handleToolCall(req.ID, req.Params)
	case "notifications/initialized":
		logger.Printf("Host confirmed handshake")
	default:
		logger.Printf("unknown method: %s", req.Method)
	}
}

func handleToolCall(id interface{}, rawMessage json.RawMessage) {
	var params mcp.CallParams
	if err := json.Unmarshal(rawMessage, &params); err != nil {
		logger.Printf("Failed to unmarshal tool call params: %v", err)
		return
	}
	switch params.Name {
	case "read_clipboard_text":
		text, err := readClipboardText()
		writeResponse(id, mcp.CallResult{
			Content: []mcp.ToolContent{
				{
					Type: "text",
					Text: text,
				},
			},
			IsError: err != nil,
		})
	case "write_to_clipboard_text":
		text, exists := params.Arguments["text_to_copy"].(string)
		if !exists {
			writeResponse(id, mcp.CallResult{
				Content: []mcp.ToolContent{},
				IsError: true,
			})
			return
		}
		writeToClipboardText(text)
		writeResponse(id, mcp.CallResult{
			Content: []mcp.ToolContent{},
			IsError: false,
		})
	case "read_clipboard_image":
		contentArray, err := readClipboardImage()
		writeResponse(id, mcp.CallResult{
			Content: contentArray,
			IsError: err != nil,
		},
		)
	case "write_to_clipboard_image":
		base64png, exists := params.Arguments["base64_png"].(string)
		if !exists {
			writeResponse(id, mcp.CallResult{
				Content: []mcp.ToolContent{},
				IsError: true,
			})
			return
		}
		writeToClipboardImage(base64png)
		writeResponse(id, mcp.CallResult{
			Content: []mcp.ToolContent{},
			IsError: false,
		})
	default:
		writeResponse(id, mcp.CallResult{
			Content: []mcp.ToolContent{},
			IsError: true,
		})
	}
}

func writeToClipboardImage(base64png string) error {
	imgBytes, err := base64.StdEncoding.DecodeString(base64png)
	if err != nil {
		return fmt.Errorf("Invalid base64 image data: %v", err)
	}
	clipboard.Write(clipboard.FmtImage, imgBytes)
	return nil
}

func readClipboardImage() ([]mcp.ToolContent, error) {
	imgBytes := clipboard.Read(clipboard.FmtImage)
	if len(imgBytes) == 0 {
		return []mcp.ToolContent{}, fmt.Errorf("clipboard is empty")
	}
	base64Str := base64.StdEncoding.EncodeToString(imgBytes)
	return []mcp.ToolContent{
		{
			Type:     "image",
			Data:     base64Str,
			MimeType: "image/png",
		},
		{
			Type: "text",
			Text: "Read image successfully from clipboard",
		},
	}, nil
}

func writeToClipboardText(text string) {
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func readClipboardText() (string, error) {
	textBytes := clipboard.Read(clipboard.FmtText)
	if len(textBytes) == 0 {
		return "", fmt.Errorf("clipboard is empty")
	}
	return string(textBytes), nil
}

func writeResponse(id interface{}, result interface{}) {
	resp := mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		logger.Printf("Failed to marshal response: %v", err)
		return
	}
	fmt.Printf("%s\n", string(bytes))
}

func handleInitialize(id interface{}) {
	result := mcp.InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: mcp.ServerInfo{
			Name:    "Clipboard-MCP",
			Version: "1.0.0",
		},
		Capabilities: mcp.ServerCapabilities{
			Tools: map[string]interface{}{},
		},
	}
	writeResponse(id, result)
	logger.Printf("Send handshake")
}
