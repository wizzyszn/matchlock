package txline

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// SSEMessage is a parsed Server-Sent Events block.
type SSEMessage struct {
	ID    string
	Event string
	Data  string
	Retry int
}

// ParseSSEBlock parses one SSE event block (lines between blank-line separators).
func ParseSSEBlock(block string) (*SSEMessage, bool) {
	if strings.TrimSpace(block) == "" {
		return nil, false
	}

	msg := &SSEMessage{}
	for _, rawLine := range strings.Split(block, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		sep := strings.Index(line, ":")
		field := line
		value := ""
		if sep >= 0 {
			field = line[:sep]
			value = line[sep+1:]
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
		}

		switch field {
		case "data":
			if msg.Data != "" {
				msg.Data += "\n"
			}
			msg.Data += value
		case "event":
			msg.Event = value
		case "id":
			msg.ID = value
		case "retry":
			// best-effort; stream reconnect uses our own backoff
			_ = value
		}
	}

	if msg.Data == "" && msg.Event == "" && msg.ID == "" {
		return nil, false
	}
	return msg, true
}

// ReadSSE reads SSE blocks from r and sends them to out until EOF or ctx cancel.
func ReadSSE(r io.Reader, out chan<- SSEMessage) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var buf bytes.Buffer
	flush := func() error {
		if buf.Len() == 0 {
			return nil
		}
		msg, ok := ParseSSEBlock(buf.String())
		buf.Reset()
		if !ok {
			return nil
		}
		out <- *msg
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}
