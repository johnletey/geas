package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// jsonrpcMessage is a JSON-RPC 2.0 message.
type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// transport handles reading and writing LSP messages over a stream.
type transport struct {
	reader *bufio.Reader
	writer io.Writer
}

func newTransport(r io.Reader, w io.Writer) *transport {
	return &transport{
		reader: bufio.NewReader(r),
		writer: w,
	}
}

// readMessage reads one LSP message from the transport.
func (t *transport) readMessage() (*jsonrpcMessage, error) {
	// Read headers.
	var contentLength int
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break // empty line separates headers from body
		}
		if strings.HasPrefix(line, "Content-Length: ") {
			val := strings.TrimPrefix(line, "Content-Length: ")
			contentLength, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read body.
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(t.reader, body); err != nil {
		return nil, err
	}

	var msg jsonrpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC message: %w", err)
	}
	return &msg, nil
}

// writeMessage writes an LSP message to the transport.
func (t *transport) writeMessage(msg *jsonrpcMessage) error {
	msg.JSONRPC = "2.0"
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(t.writer, header); err != nil {
		return err
	}
	_, err = t.writer.Write(body)
	return err
}

// sendResult sends a successful response.
func (t *transport) sendResult(id *json.RawMessage, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return t.writeMessage(&jsonrpcMessage{
		ID:     id,
		Result: data,
	})
}

// sendError sends an error response.
func (t *transport) sendError(id *json.RawMessage, code int, message string) error {
	return t.writeMessage(&jsonrpcMessage{
		ID:    id,
		Error: &jsonrpcError{Code: code, Message: message},
	})
}

// sendNotification sends a notification (no id).
func (t *transport) sendNotification(method string, params any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return t.writeMessage(&jsonrpcMessage{
		Method: method,
		Params: data,
	})
}
