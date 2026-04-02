package log

import "testing"

func TestCLF_Combined(t *testing.T) {
	f := &CLFFormat{}
	line := `127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326 "http://www.example.com/start.html" "Mozilla/4.08"`
	rec, err := f.ParseRecord(line)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Msg != "GET /apache_pb.gif HTTP/1.0" {
		t.Errorf("msg = %q", rec.Msg)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if rec.Time.Year() != 2000 {
		t.Errorf("year = %d, want 2000", rec.Time.Year())
	}
	if rec.Attrs["remote_addr"] != "127.0.0.1" {
		t.Errorf("remote_addr = %v", rec.Attrs["remote_addr"])
	}
	if rec.Attrs["user"] != "frank" {
		t.Errorf("user = %v", rec.Attrs["user"])
	}
	if rec.Attrs["status"] != float64(200) {
		t.Errorf("status = %v, want 200", rec.Attrs["status"])
	}
	if rec.Attrs["bytes"] != float64(2326) {
		t.Errorf("bytes = %v, want 2326", rec.Attrs["bytes"])
	}
	if rec.Attrs["referer"] != "http://www.example.com/start.html" {
		t.Errorf("referer = %v", rec.Attrs["referer"])
	}
}

func TestCLF_CommonOnly(t *testing.T) {
	f := &CLFFormat{}
	line := `10.0.0.1 - - [15/Jan/2024:10:30:00 +0000] "POST /api/data HTTP/1.1" 201 512`
	rec, err := f.ParseRecord(line)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Msg != "POST /api/data HTTP/1.1" {
		t.Errorf("msg = %q", rec.Msg)
	}
	if rec.Level != "INFO" {
		t.Errorf("level = %q, want INFO", rec.Level)
	}
	if _, ok := rec.Attrs["user"]; ok {
		t.Error("user should not be set for '-'")
	}
}

func TestCLF_ServerError(t *testing.T) {
	f := &CLFFormat{}
	line := `10.0.0.1 - - [15/Jan/2024:10:30:00 +0000] "GET /fail HTTP/1.1" 500 0 "-" "curl/7.0"`
	rec, err := f.ParseRecord(line)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", rec.Level)
	}
}

func TestCLF_ClientError(t *testing.T) {
	f := &CLFFormat{}
	line := `10.0.0.1 - - [15/Jan/2024:10:30:00 +0000] "GET /missing HTTP/1.1" 404 0 "-" "curl/7.0"`
	rec, err := f.ParseRecord(line)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Level != "WARN" {
		t.Errorf("level = %q, want WARN", rec.Level)
	}
}

func TestCLF_RequestLineParsed(t *testing.T) {
	f := &CLFFormat{}
	line := `10.0.0.1 - - [15/Jan/2024:10:30:00 +0000] "GET /api/v1/users HTTP/1.1" 200 512 "-" "curl/7.0"`
	rec, err := f.ParseRecord(line)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Attrs["method"] != "GET" {
		t.Errorf("method = %v, want GET", rec.Attrs["method"])
	}
	if rec.Attrs["path"] != "/api/v1/users" {
		t.Errorf("path = %v, want /api/v1/users", rec.Attrs["path"])
	}
	if rec.Attrs["protocol"] != "HTTP/1.1" {
		t.Errorf("protocol = %v, want HTTP/1.1", rec.Attrs["protocol"])
	}
}

func TestCLF_QueryParams(t *testing.T) {
	f := &CLFFormat{}
	line := `10.0.0.1 - - [15/Jan/2024:10:30:00 +0000] "GET /search?q=hello&page=2 HTTP/1.1" 200 512 "-" "curl/7.0"`
	rec, err := f.ParseRecord(line)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Attrs["path"] != "/search" {
		t.Errorf("path = %v, want /search", rec.Attrs["path"])
	}
	if rec.Attrs["query.q"] != "hello" {
		t.Errorf("query.q = %v, want hello", rec.Attrs["query.q"])
	}
	if rec.Attrs["query.page"] != "2" {
		t.Errorf("query.page = %v, want 2", rec.Attrs["query.page"])
	}
}

func TestCLF_Invalid(t *testing.T) {
	f := &CLFFormat{}
	_, err := f.ParseRecord("not a log line")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCLF_Name(t *testing.T) {
	f := &CLFFormat{}
	if f.Name() != "clf" {
		t.Errorf("name = %q, want clf", f.Name())
	}
}
