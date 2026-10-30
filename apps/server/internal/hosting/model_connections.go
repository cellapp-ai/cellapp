package hosting

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type modelConnectionView struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	APIBase      string          `json:"apiBase"`
	ModelID      string          `json:"modelId"`
	Capabilities json.RawMessage `json:"capabilities"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

func validModelConnection(name, base, model, key string) error {
	if strings.TrimSpace(name) != name || name == "" || utf8.RuneCountInString(name) > 80 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fail(400, "invalid_model_connection", "Enter a valid connection name")
	}
	if len(model) > 200 || strings.TrimSpace(model) != model || strings.IndexFunc(model, unicode.IsControl) >= 0 {
		return fail(400, "invalid_model_connection", "Enter a valid model ID")
	}
	if len(key) < 1 || len(key) > 4096 || strings.TrimSpace(key) != key || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return fail(400, "invalid_model_connection", "Enter a valid API key")
	}
	if _, err := modelChatURL(base); err != nil {
		return err
	}
	return nil
}

func (s *Server) modelCipher() (cipher.AEAD, error) {
	if len(s.Config.ModelEncryptionKey) != 32 {
		return nil, fail(503, "model_secret_unavailable", "Model connections are unavailable")
	}
	block, err := aes.NewCipher(s.Config.ModelEncryptionKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Server) encryptModelKey(owner, connection, key string) ([]byte, error) {
	aead, err := s.modelCipher()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, []byte(key), []byte(owner+":"+connection)), nil
}

func (s *Server) decryptModelKey(owner, connection string, encrypted []byte, version int) (string, error) {
	if version != 1 {
		return "", fmt.Errorf("unsupported model key version")
	}
	aead, err := s.modelCipher()
	if err != nil {
		return "", err
	}
	if len(encrypted) < aead.NonceSize() {
		return "", fmt.Errorf("invalid model key ciphertext")
	}
	plain, err := aead.Open(nil, encrypted[:aead.NonceSize()], encrypted[aead.NonceSize():], []byte(owner+":"+connection))
	if err != nil {
		return "", fmt.Errorf("model key decryption failed")
	}
	return string(plain), nil
}

func (s *Server) webModelConnections(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, err := s.DB.Query(r.Context(), `SELECT id,name,api_base,model_id,capabilities,created_at,updated_at
	FROM model_connections WHERE owner_id=$1 ORDER BY created_at DESC,id`, owner)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []modelConnectionView{}
	for rows.Next() {
		var item modelConnectionView
		if err := rows.Scan(&item.ID, &item.Name, &item.APIBase, &item.ModelID, &item.Capabilities, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	jsonResponse(w, 200, items)
	return nil
}

func (s *Server) webCreateModelConnection(w http.ResponseWriter, r *http.Request, owner string) error {
	var input struct {
		Name, APIBase, ModelID, APIKey string
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if err := validModelConnection(input.Name, input.APIBase, input.ModelID, input.APIKey); err != nil {
		return err
	}
	if input.ModelID == "" {
		models := s.providerModels(r.Context(), catalogConnection{base: input.APIBase, key: input.APIKey})
		for _, model := range models {
			if model != "" && len(model) <= 200 && strings.TrimSpace(model) == model && strings.IndexFunc(model, unicode.IsControl) < 0 {
				input.ModelID = model
				break
			}
		}
		if input.ModelID == "" {
			return fail(400, "model_discovery_failed", "Provider model listing failed; enter a model ID manually")
		}
	}
	connection := id()
	encrypted, err := s.encryptModelKey(owner, connection, input.APIKey)
	if err != nil {
		return err
	}
	var item modelConnectionView
	err = s.DB.QueryRow(r.Context(), `INSERT INTO model_connections(id,owner_id,name,api_base,model_id,encrypted_key,key_version)
	VALUES($1,$2,$3,$4,$5,$6,1) RETURNING id,name,api_base,model_id,capabilities,created_at,updated_at`,
		connection, owner, input.Name, input.APIBase, input.ModelID, encrypted).Scan(&item.ID, &item.Name, &item.APIBase, &item.ModelID, &item.Capabilities, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return err
	}
	jsonResponse(w, 201, item)
	return nil
}

func (s *Server) webRotateModelKey(w http.ResponseWriter, r *http.Request, owner string) error {
	var input struct {
		APIKey string
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if len(input.APIKey) < 1 || len(input.APIKey) > 4096 || strings.TrimSpace(input.APIKey) != input.APIKey || strings.IndexFunc(input.APIKey, unicode.IsControl) >= 0 {
		return fail(400, "invalid_model_key", "Enter a valid API key")
	}
	connection := r.PathValue("connection")
	if !idPattern.MatchString(connection) {
		return fail(404, "model_connection_not_found", "Model connection not found")
	}
	encrypted, err := s.encryptModelKey(owner, connection, input.APIKey)
	if err != nil {
		return err
	}
	result, err := s.DB.Exec(r.Context(), `UPDATE model_connections SET encrypted_key=$1,key_version=1,capabilities='{}'::jsonb,updated_at=now()
	WHERE id=$2 AND owner_id=$3`, encrypted, connection, owner)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(404, "model_connection_not_found", "Model connection not found")
	}
	jsonResponse(w, 200, map[string]bool{"rotated": true})
	return nil
}

func (s *Server) webDeleteModelConnection(w http.ResponseWriter, r *http.Request, owner string) error {
	connection := r.PathValue("connection")
	if !idPattern.MatchString(connection) {
		return fail(404, "model_connection_not_found", "Model connection not found")
	}
	result, err := s.DB.Exec(r.Context(), `DELETE FROM model_connections WHERE id=$1 AND owner_id=$2`, connection, owner)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(404, "model_connection_not_found", "Model connection not found")
	}
	jsonResponse(w, 200, map[string]bool{"deleted": true})
	return nil
}

func (s *Server) modelKeyForOwner(r *http.Request, owner, connection string) (string, error) {
	var encrypted []byte
	var version int
	err := s.DB.QueryRow(r.Context(), `SELECT encrypted_key,key_version FROM model_connections WHERE id=$1 AND owner_id=$2`, connection, owner).Scan(&encrypted, &version)
	if err == pgx.ErrNoRows {
		return "", fail(404, "model_connection_not_found", "Model connection not found")
	}
	if err != nil {
		return "", err
	}
	return s.decryptModelKey(owner, connection, encrypted, version)
}
