package nginx

import "testing"

func TestParseProcStat(t *testing.T) {
	// field 2 name contains spaces and a ')'
	data := []byte("123 (nginx: worker process) S 1 123 0 0 -1 4194304 100 0 0 0 50 30 0 0 20 0 1 0 12345 0 0 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0")
	utime, stime, ok := parseProcStat(data)
	if !ok {
		t.Fatal("expected ok")
	}
	if utime != 50 || stime != 30 {
		t.Fatalf("expected utime=50 stime=30, got %d %d", utime, stime)
	}
}

func TestParseVmRSS(t *testing.T) {
	data := []byte("Name:\tnginx\nVmRSS:\t   123456 kB\nVmSize:\t987654 kB\n")
	if got := parseVmRSS(data); got != 123456 {
		t.Fatalf("expected 123456, got %d", got)
	}
	if got := parseVmRSS([]byte("no rss here")); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}
