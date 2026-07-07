package txline

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSSEBlockAllFields(t *testing.T) {
	block := "id: evt-1\nevent: score\nretry: 5000\ndata: {\"fixtureId\":1}\n"
	msg, ok := ParseSSEBlock(block)
	if !ok || msg.ID != "evt-1" || msg.Event != "score" || msg.Data == "" {
		t.Fatalf("msg = %#v ok=%v", msg, ok)
	}
}

func TestParseSSEBlockFieldWithoutColon(t *testing.T) {
	_, ok := ParseSSEBlock("data\n")
	if ok {
		t.Fatal("expected empty block to be rejected")
	}
}

type failReader struct{}

func (f *failReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (f *failReader) Close() error { return nil }

func TestReadSSEMultiLineData(t *testing.T) {
	input := strings.NewReader("event: score\ndata: line1\ndata: line2\n\n")
	out := make(chan SSEMessage, 1)
	if err := ReadSSE(input, out); err != nil {
		t.Fatalf("ReadSSE: %v", err)
	}
	close(out)
	msg := <-out
	if msg.Data != "line1\nline2" {
		t.Fatalf("data = %q", msg.Data)
	}
}

func TestReadSSEFlushesTrailingBlock(t *testing.T) {
	input := strings.NewReader("event: score\ndata: one\n")
	out := make(chan SSEMessage, 1)
	if err := ReadSSE(input, out); err != nil {
		t.Fatalf("ReadSSE: %v", err)
	}
	close(out)
	msg := <-out
	if msg.Event != "score" || msg.Data != "one" {
		t.Fatalf("msg = %#v", msg)
	}
}