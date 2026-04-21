package images

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/AndroidPoet/playconsole-cli/internal/output"
)

func TestProgressReaderReportsAtMostOncePerBucket(t *testing.T) {
	var buf bytes.Buffer
	originalWriter := os.Stdout
	output.SetWriter(&buf)
	t.Cleanup(func() {
		output.SetWriter(originalWriter)
	})

	reader := newProgressReader(bytes.NewReader(make([]byte, 1000)), 1000, "demo.png")
	chunk := make([]byte, 100)

	for {
		_, err := reader.Read(chunk)
		if err != nil {
			break
		}
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if got, want := len(lines), 10; got != want {
		t.Fatalf("line count = %d, want %d\n%s", got, want, buf.String())
	}
	if lines[0] != "Upload progress: demo.png 10% (100 B/1000 B)" {
		t.Fatalf("first line = %q", lines[0])
	}
	if lines[len(lines)-1] != "Upload progress: demo.png 100% (1000 B/1000 B)" {
		t.Fatalf("last line = %q", lines[len(lines)-1])
	}
}

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		size int64
		want string
	}{
		{size: 999, want: "999 B"},
		{size: 1024, want: "1.0 KiB"},
		{size: 1536, want: "1.5 KiB"},
		{size: 1024 * 1024, want: "1.0 MiB"},
	}

	for _, tc := range testCases {
		if got := formatBytes(tc.size); got != tc.want {
			t.Fatalf("formatBytes(%d) = %q, want %q", tc.size, got, tc.want)
		}
	}
}
