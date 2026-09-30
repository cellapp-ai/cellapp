package hosting

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestS3ConfigAndBucketHealth(t *testing.T) {
	calls := 0
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "HEAD" || r.URL.Path != "/private-bucket/" {
			t.Errorf("unexpected bucket probe %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("unsigned storage request")
		}
		w.WriteHeader(200)
	}))
	defer endpoint.Close()
	c, _ := LoadConfig(func(string) string { return "" })
	c.S3Endpoint = endpoint.URL
	c.S3Bucket = "private-bucket"
	storage, e := NewStorage(c)
	if e != nil {
		t.Fatal(e)
	}
	if e = storage.Health(context.Background()); e != nil {
		t.Fatal(e)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	c.Production = true
	if _, e = NewStorage(c); e == nil {
		t.Fatal("insecure production endpoint accepted")
	}
}

func TestS3SignedObjectRoundTrip(t *testing.T) {
	var stored []byte
	endpoint := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("unsigned object operation")
			w.WriteHeader(403)
			return
		}
		if r.URL.Path != "/private-bucket/releases/test" {
			t.Errorf("wrong object path: %s", r.URL.Path)
		}
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		switch r.Method {
		case "PUT":
			var e error
			stored, e = io.ReadAll(r.Body)
			if e != nil {
				t.Error(e)
			}
			w.Header().Set("ETag", `"abc"`)
			w.WriteHeader(200)
		case "HEAD":
			if stored == nil {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", "5")
			w.Header().Set("ETag", `"abc"`)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(200)
		case "GET":
			if stored == nil {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", "5")
			w.Header().Set("ETag", `"abc"`)
			_, _ = w.Write(stored)
		case "DELETE":
			stored = nil
			w.WriteHeader(204)
		default:
			t.Error("unexpected S3 operation", r.Method)
			w.WriteHeader(405)
		}
	}))
	defer endpoint.Close()
	u, _ := url.Parse(endpoint.URL)
	client, e := minio.New(u.Host, &minio.Options{Creds: credentials.NewStaticV4("test-ak", "test-sk", ""), Secure: true, Region: "cn-east-1", BucketLookup: minio.BucketLookupPath, Transport: endpoint.Client().Transport})
	if e != nil {
		t.Fatal(e)
	}
	storage := &S3Storage{client: client, bucket: "private-bucket"}
	ctx := context.Background()
	if e = storage.Put(ctx, "releases/test", bytes.NewReader([]byte("hello")), 5); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(stored, []byte("hello")) {
		t.Fatal("unexpected S3 payload format")
	}
	reader, e := storage.Get(ctx, "releases/test")
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(reader)
	_ = reader.Close()
	if e != nil || string(got) != "hello" {
		t.Fatal(e, string(got))
	}
	if e = storage.Delete(ctx, "releases/test"); e != nil {
		t.Fatal(e)
	}
	if stored != nil {
		t.Fatal("object not deleted")
	}
}

func TestLiveS3(t *testing.T) {
	if os.Getenv("RUN_S3_TESTS") != "1" {
		t.Skip("Qiniu bucket and credentials deferred; set RUN_S3_TESTS=1 explicitly after configuration")
	}
	c, e := LoadConfig(os.Getenv)
	if e != nil {
		t.Fatal(e)
	}
	storage, e := NewStorage(c)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if e = storage.Health(ctx); e != nil {
		t.Fatal(e)
	}
	key := "cellapp-probes/" + id()
	body := []byte("cellapp-private-storage-probe")
	if e = storage.Put(ctx, key, bytes.NewReader(body), int64(len(body))); e != nil {
		t.Fatal(e)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if e := storage.Delete(cleanup, key); e != nil {
			t.Errorf("probe cleanup failed: %v", e)
		}
	}()
	object, e := storage.Get(ctx, key)
	if e != nil {
		t.Fatal(e)
	}
	got, e := io.ReadAll(object)
	_ = object.Close()
	if e != nil || !bytes.Equal(got, body) {
		t.Fatal("object read mismatch", e)
	}
	req, e := http.NewRequestWithContext(ctx, "GET", c.S3Endpoint+"/"+c.S3Bucket+"/"+key, nil)
	if e != nil {
		t.Fatal(e)
	}
	response, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	anonymousBody, e := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if e != nil {
		t.Fatal(e)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 || bytes.Contains(anonymousBody, body) {
		t.Fatalf("anonymous request exposed object content with status %d", response.StatusCode)
	}
	if e = storage.Delete(ctx, key); e != nil {
		t.Fatal(e)
	}
	if object, e = storage.Get(ctx, key); e == nil {
		_ = object.Close()
		t.Fatal("deleted probe remains readable")
	}
}
