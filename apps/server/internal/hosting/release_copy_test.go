package hosting

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type interruptedStorage struct {
	*memoryStorage
	failOnce bool
	puts     int
}

func (s *interruptedStorage) Put(ctx context.Context, key string, body io.Reader, size int64) error {
	s.puts++
	if s.failOnce {
		s.failOnce = false
		return errors.New("interrupted copy")
	}
	return s.memoryStorage.Put(ctx, key, body, size)
}

func TestCopyReleaseIntegrityRetryAndMissingObject(t *testing.T) {
	ctx := context.Background()
	deployment := strings.Repeat("d", 32)
	content := []byte("hello release")
	file := Entry{Path: "index.html", Size: int64(len(content)), SHA256: digest(content)}
	source := &memoryStorage{data: map[string][]byte{objectKey(deployment, file.Path): content}}
	destination := &interruptedStorage{memoryStorage: &memoryStorage{data: map[string][]byte{}}, failOnce: true}
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, false); err == nil {
		t.Fatal("interrupted copy reported success")
	}
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, false); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if destination.data[releaseManifestKey(deployment)] == nil {
		t.Fatal("release manifest not copied")
	}
	puts := destination.puts
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, false); err != nil || destination.puts != puts {
		t.Fatalf("completed file not resumed: puts=%d err=%v", destination.puts, err)
	}
	destination.data[objectKey(deployment, file.Path)] = []byte("corrupted")
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, true); err == nil {
		t.Fatal("verify-only accepted corrupted destination")
	}
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, false); err != nil {
		t.Fatalf("corrupted destination was not repaired: %v", err)
	}
	source.data[objectKey(deployment, file.Path)] = []byte("corrupted source")
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, false); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("source corruption not detected: %v", err)
	}
	delete(source.data, objectKey(deployment, file.Path))
	if err := CopyRelease(ctx, source, destination, deployment, []Entry{file}, false); err == nil {
		t.Fatal("missing source object accepted")
	}
}

func TestCopyPublishedReleasesIncludesRollbackVersion(t *testing.T) {
	f := setup(t)
	app, _ := f.create(t)
	old, _ := f.stage(t, app, "old")
	f.uploadAndPublish(t, app, old, "old")
	app.Active = &old
	active, _ := f.stage(t, app, "new")
	f.uploadAndPublish(t, app, active, "new")
	destination := &memoryStorage{data: map[string][]byte{}}
	count, err := CopyPublishedReleases(context.Background(), f.db, f.storage, destination, 24*time.Hour, false)
	if err != nil || count != 2 {
		t.Fatalf("rollback releases copied=%d err=%v", count, err)
	}
	for _, deployment := range []string{old, active} {
		if !bytes.Contains(destination.data[releaseManifestKey(deployment)], []byte("index.html")) {
			t.Fatalf("release %s missing manifest", deployment)
		}
		if !bytes.Contains(f.storage.data[releaseManifestKey(deployment)], []byte("index.html")) {
			t.Fatalf("rollback source release %s changed during copy", deployment)
		}
	}
	count, err = CopyPublishedReleases(context.Background(), f.db, f.storage, destination, 24*time.Hour, true)
	if err != nil || count != 2 {
		t.Fatalf("rollback verification copied=%d err=%v", count, err)
	}
}
