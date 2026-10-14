package hosting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
)

func missingObject(err error) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	var response minio.ErrorResponse
	return errors.As(err, &response) && (response.Code == "NoSuchKey" || response.Code == "NoSuchObject" || response.Code == "NotFound")
}

func verifyStoredObject(ctx context.Context, storage Storage, key, expectedHash string, expectedSize int64) (bool, error) {
	object, err := storage.Get(ctx, key)
	if err != nil {
		if missingObject(err) {
			return false, nil
		}
		return false, err
	}
	defer object.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, object)
	if err != nil {
		return false, err
	}
	return size == expectedSize && hex.EncodeToString(hash.Sum(nil)) == expectedHash, nil
}

func ensureObject(ctx context.Context, destination Storage, key string, source io.ReadSeeker, size int64, checksum string, verifyOnly bool) error {
	valid, err := verifyStoredObject(ctx, destination, key, checksum, size)
	if err != nil {
		return err
	}
	if valid {
		return nil
	}
	if verifyOnly {
		return fmt.Errorf("destination object missing or corrupted: %s", key)
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := destination.Put(ctx, key, source, size); err != nil {
		return err
	}
	valid, err = verifyStoredObject(ctx, destination, key, checksum, size)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("destination verification failed: %s", key)
	}
	return nil
}

// CopyRelease verifies source bytes against the immutable database manifest,
// resumes completed files, and writes edge metadata only after every file passes.
func CopyRelease(ctx context.Context, source, destination Storage, deployment string, files []Entry, verifyOnly bool) error {
	for _, file := range files {
		key := objectKey(deployment, file.Path)
		object, err := source.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("source object %s: %w", key, err)
		}
		temp, err := os.CreateTemp("", "cellapp-release-copy-*")
		if err != nil {
			object.Close()
			return err
		}
		hash := sha256.New()
		size, copyErr := io.Copy(io.MultiWriter(temp, hash), object)
		closeErr := object.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		if copyErr == nil && (size != file.Size || hex.EncodeToString(hash.Sum(nil)) != file.SHA256) {
			copyErr = fmt.Errorf("source digest mismatch: %s", key)
		}
		if copyErr == nil {
			copyErr = ensureObject(ctx, destination, key, temp, size, file.SHA256, verifyOnly)
		}
		temp.Close()
		os.Remove(temp.Name())
		if copyErr != nil {
			return copyErr
		}
	}
	manifest, err := json.Marshal(struct {
		Files []Entry `json:"files"`
	}{Files: files})
	if err != nil {
		return err
	}
	sum := sha256.Sum256(manifest)
	return ensureObject(ctx, destination, releaseManifestKey(deployment), bytes.NewReader(manifest), int64(len(manifest)), hex.EncodeToString(sum[:]), verifyOnly)
}

// CopyPublishedReleases includes the active release and every published release
// still within the rollback retention window. It never deletes source objects.
func CopyPublishedReleases(ctx context.Context, db *pgxpool.Pool, source, destination Storage, retention time.Duration, verifyOnly bool) (int, error) {
	rows, err := db.Query(ctx, `SELECT d.id,d.manifest FROM deployments d JOIN apps a ON a.id=d.app_id
		WHERE d.status='published' AND (d.id=a.active_deployment OR d.published_at >= $1)
		ORDER BY d.id`, time.Now().Add(-retention))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var deployment string
		var raw []byte
		if err := rows.Scan(&deployment, &raw); err != nil {
			return count, err
		}
		var files []Entry
		if err := json.Unmarshal(raw, &files); err != nil {
			return count, err
		}
		if err := CopyRelease(ctx, source, destination, deployment, files, verifyOnly); err != nil {
			return count, fmt.Errorf("release %s: %w", deployment, err)
		}
		count++
	}
	return count, rows.Err()
}
