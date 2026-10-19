package hosting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const mcpChunkBytes = 256 << 10

type mcpChunkInput struct {
	AppID        string `json:"appId"`
	DeploymentID string `json:"deploymentId"`
	FileIndex    int    `json:"fileIndex"`
	OffsetBytes  int64  `json:"offsetBytes"`
	SizeBytes    int    `json:"sizeBytes"`
	SHA256       string `json:"sha256"`
	DataBase64   string `json:"dataBase64"`
}

func mcpChunkKey(deploymentID string, fileIndex int, offset int64) string {
	return fmt.Sprintf("mcp-chunks/%s/%d/%d", deploymentID, fileIndex, offset)
}

func (s *Server) uploadMCPChunk(ctx context.Context, owner string, in mcpChunkInput) (int64, error) {
	if s.Config.PublicationPaused {
		return 0, fail(503, "publication_paused", "Application publication is temporarily paused")
	}
	if !idPattern.MatchString(in.AppID) || !idPattern.MatchString(in.DeploymentID) || in.FileIndex < 0 || in.OffsetBytes < 0 || in.SizeBytes <= 0 || in.SizeBytes > mcpChunkBytes || !digestPattern.MatchString(in.SHA256) || len(in.DataBase64) > base64.StdEncoding.EncodedLen(mcpChunkBytes) {
		return 0, fail(400, "invalid_chunk", "Invalid chunk metadata")
	}
	data, err := base64.StdEncoding.DecodeString(in.DataBase64)
	if err != nil || len(data) != in.SizeBytes || digest(data) != in.SHA256 {
		return 0, fail(400, "digest_mismatch", "Chunk does not match declared size and digest")
	}
	var next int64
	err = s.transaction(ctx, func(tx pgx.Tx) error {
		app, err := s.app(ctx, tx, in.AppID, owner)
		if err != nil {
			return err
		}
		if app.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		if s.AppDataDB != nil && app.DataState != "ready" {
			return fail(503, "app_data_not_ready", "Application data is not ready")
		}
		// Serialize chunks for this deployment before touching their shared
		// object keys. A losing concurrent offset must never replace the object
		// referenced by the winning database row.
		if _, err := tx.Exec(ctx, `SELECT id FROM deployments WHERE id=$1 FOR UPDATE`, in.DeploymentID); err != nil {
			return err
		}
		deployment, err := getDeployment(ctx, tx, in.AppID, in.DeploymentID)
		if err != nil {
			return err
		}
		if deployment.Status != "uploading" || time.Since(deployment.Created) > s.Config.UploadTTL {
			return fail(409, "deployment_closed", "Start a new deployment")
		}
		if in.FileIndex >= len(deployment.Manifest) {
			return fail(400, "invalid_index", "Invalid file index")
		}
		file := deployment.Manifest[in.FileIndex]
		if in.OffsetBytes > file.Size || int64(in.SizeBytes) > file.Size-in.OffsetBytes {
			return fail(400, "invalid_chunk", "Chunk exceeds manifest file")
		}
		for _, uploaded := range deployment.Uploaded {
			if uploaded == file.Path {
				return fail(409, "file_complete", "File is already complete")
			}
		}
		var existingSize int
		var existingDigest string
		err = tx.QueryRow(ctx, `SELECT size_bytes,sha256 FROM mcp_upload_chunks WHERE deployment_id=$1 AND file_index=$2 AND offset_bytes=$3`, in.DeploymentID, in.FileIndex, in.OffsetBytes).Scan(&existingSize, &existingDigest)
		if err == nil {
			if existingSize != in.SizeBytes || existingDigest != in.SHA256 {
				return fail(409, "chunk_conflict", "Offset already contains another chunk")
			}
			if err := s.Storage.Put(ctx, mcpChunkKey(in.DeploymentID, in.FileIndex, in.OffsetBytes), bytes.NewReader(data), int64(len(data))); err != nil {
				return err
			}
			next = in.OffsetBytes + int64(in.SizeBytes)
			return nil
		}
		if err != pgx.ErrNoRows {
			return err
		}
		var current int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(size_bytes),0) FROM mcp_upload_chunks WHERE deployment_id=$1 AND file_index=$2`, in.DeploymentID, in.FileIndex).Scan(&current); err != nil {
			return err
		}
		if in.OffsetBytes != current {
			return fail(409, "chunk_offset", "Resume at the reported next offset")
		}
		key := mcpChunkKey(in.DeploymentID, in.FileIndex, in.OffsetBytes)
		if err := s.Storage.Put(ctx, key, bytes.NewReader(data), int64(len(data))); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO mcp_upload_chunks(deployment_id,file_index,offset_bytes,size_bytes,sha256,object_key) VALUES($1,$2,$3,$4,$5,$6)`, in.DeploymentID, in.FileIndex, in.OffsetBytes, in.SizeBytes, in.SHA256, key); err != nil {
			return err
		}
		next = current + int64(in.SizeBytes)
		return nil
	})
	return next, err
}

type mcpChunkRecord struct {
	offset int64
	size   int64
	key    string
}

func (s *Server) completeMCPFile(ctx context.Context, owner, appID, deploymentID string, fileIndex int) error {
	if s.Config.PublicationPaused {
		return fail(503, "publication_paused", "Application publication is temporarily paused")
	}
	if !idPattern.MatchString(appID) || !idPattern.MatchString(deploymentID) || fileIndex < 0 {
		return fail(400, "invalid_index", "Invalid file index")
	}
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		app, err := s.app(ctx, tx, appID, owner)
		if err != nil {
			return err
		}
		if app.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		if s.AppDataDB != nil && app.DataState != "ready" {
			return fail(503, "app_data_not_ready", "Application data is not ready")
		}
		if _, err := tx.Exec(ctx, `SELECT id FROM deployments WHERE id=$1 FOR UPDATE`, deploymentID); err != nil {
			return err
		}
		deployment, err := getDeployment(ctx, tx, appID, deploymentID)
		if err != nil {
			return err
		}
		if deployment.Status != "uploading" || time.Since(deployment.Created) > s.Config.UploadTTL {
			return fail(409, "deployment_closed", "Start a new deployment")
		}
		if fileIndex >= len(deployment.Manifest) {
			return fail(400, "invalid_index", "Invalid file index")
		}
		file := deployment.Manifest[fileIndex]
		for _, uploaded := range deployment.Uploaded {
			if uploaded == file.Path {
				return nil
			}
		}
		rows, err := tx.Query(ctx, `SELECT offset_bytes,size_bytes,object_key FROM mcp_upload_chunks WHERE deployment_id=$1 AND file_index=$2 ORDER BY offset_bytes`, deploymentID, fileIndex)
		if err != nil {
			return err
		}
		chunks := []mcpChunkRecord{}
		var total int64
		for rows.Next() {
			var chunk mcpChunkRecord
			if err := rows.Scan(&chunk.offset, &chunk.size, &chunk.key); err != nil {
				rows.Close()
				return err
			}
			if chunk.offset != total {
				rows.Close()
				return fail(409, "incomplete_upload", "Missing file chunk")
			}
			total += chunk.size
			chunks = append(chunks, chunk)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if total != file.Size {
			return fail(409, "incomplete_upload", "Upload all file chunks before completion")
		}
		reader, writer := io.Pipe()
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		checksum := sha256.New()
		streamResult := make(chan error, 1)
		go func() {
			streamErr := s.streamMCPChunks(streamCtx, writer, checksum, chunks)
			writer.CloseWithError(streamErr)
			streamResult <- streamErr
		}()
		key := objectKey(deploymentID, file.Path)
		putErr := s.Storage.Put(streamCtx, key, reader, file.Size)
		if putErr != nil {
			cancel()
			reader.CloseWithError(putErr)
		}
		streamErr := <-streamResult
		reader.Close()
		if putErr != nil || streamErr != nil {
			_ = s.Storage.Delete(ctx, key)
			if streamErr != nil {
				return fail(503, "chunk_unavailable", "Chunk storage unavailable; retry file completion")
			}
			return putErr
		}
		if hex.EncodeToString(checksum.Sum(nil)) != file.SHA256 {
			_ = s.Storage.Delete(ctx, key)
			return fail(400, "digest_mismatch", "File does not match manifest")
		}
		deployment.Uploaded = append(deployment.Uploaded, file.Path)
		_, err = tx.Exec(ctx, `UPDATE deployments SET uploaded=$2 WHERE id=$1`, deploymentID, encoded(deployment.Uploaded))
		return err
	})
	if err != nil {
		return err
	}
	// The final object is committed before temporary chunks are removed. Cleanup
	// is retried by maintenance if storage is briefly unavailable.
	if err := s.deleteMCPChunksForFile(ctx, deploymentID, fileIndex); err != nil {
		s.Logger.Warn("mcp_chunk_cleanup_failed", "deployment_id", deploymentID, "file_index", fileIndex)
	}
	return nil
}

func (s *Server) deleteMCPChunksForFile(ctx context.Context, deploymentID string, fileIndex int) error {
	rows, err := s.DB.Query(ctx, `SELECT offset_bytes,object_key FROM mcp_upload_chunks WHERE deployment_id=$1 AND file_index=$2 ORDER BY offset_bytes`, deploymentID, fileIndex)
	if err != nil {
		return err
	}
	type stored struct {
		offset int64
		key    string
	}
	chunks := []stored{}
	for rows.Next() {
		var chunk stored
		if err := rows.Scan(&chunk.offset, &chunk.key); err != nil {
			rows.Close()
			return err
		}
		chunks = append(chunks, chunk)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if err := s.Storage.Delete(ctx, chunk.key); err != nil {
			return err
		}
		if _, err := s.DB.Exec(ctx, `DELETE FROM mcp_upload_chunks WHERE deployment_id=$1 AND file_index=$2 AND offset_bytes=$3`, deploymentID, fileIndex, chunk.offset); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) cleanupCompletedMCPChunks(ctx context.Context) error {
	rows, err := s.DB.Query(ctx, `SELECT DISTINCT c.deployment_id,c.file_index FROM mcp_upload_chunks c JOIN deployments d ON d.id=c.deployment_id WHERE d.status='cleaned' OR d.uploaded ? (d.manifest->c.file_index->>'path')`)
	if err != nil {
		return err
	}
	type file struct {
		deployment string
		index      int
	}
	files := []file{}
	for rows.Next() {
		var item file
		if err := rows.Scan(&item.deployment, &item.index); err != nil {
			rows.Close()
			return err
		}
		files = append(files, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range files {
		if err := s.deleteMCPChunksForFile(ctx, item.deployment, item.index); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) cleanupOrphanMCPChunks(ctx context.Context) error {
	walker, ok := s.Storage.(temporaryObjectWalker)
	if !ok {
		return nil
	}
	cutoff := time.Now().Add(-s.Config.UploadTTL)
	return walker.WalkTemporary(ctx, func(key string, modified time.Time) error {
		if !strings.HasPrefix(key, "mcp-chunks/") || modified.After(cutoff) {
			return nil
		}
		var known bool
		if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mcp_upload_chunks WHERE object_key=$1)`, key).Scan(&known); err != nil {
			return err
		}
		if known {
			return nil
		}
		return s.Storage.Delete(ctx, key)
	})
}

func (s *Server) streamMCPChunks(ctx context.Context, writer io.Writer, checksum hash.Hash, chunks []mcpChunkRecord) error {
	for _, chunk := range chunks {
		item, err := s.Storage.Get(ctx, chunk.key)
		if err != nil {
			return err
		}
		_, err = io.CopyN(io.MultiWriter(writer, checksum), item, chunk.size)
		if err == nil {
			var extra [1]byte
			count, readErr := item.Read(extra[:])
			if count != 0 || readErr != io.EOF {
				err = errors.New("chunk object size mismatch")
			}
		}
		closeErr := item.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func (s *Server) mcpChunkProgress(ctx context.Context, deploymentID string) (map[int]int64, error) {
	rows, err := s.DB.Query(ctx, `SELECT file_index,COALESCE(sum(size_bytes),0) FROM mcp_upload_chunks WHERE deployment_id=$1 GROUP BY file_index`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	progress := map[int]int64{}
	for rows.Next() {
		var index int
		var bytes int64
		if err := rows.Scan(&index, &bytes); err != nil {
			return nil, err
		}
		progress[index] = bytes
	}
	return progress, rows.Err()
}
