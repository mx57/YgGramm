// File: core/pkg/backup/snapshot_engine.go
package backup

import (
	"bytes"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	MinChunkSize    = 64 * 1024        // 64 KB
	TargetChunkSize = 256 * 1024       // 256 KB
	MaxChunkSize    = 1024 * 1024      // 1 MB
	FastCDCMaskS    = 0x0000d90f03530000 // Cutpoint mask for small chunks
	FastCDCMaskL    = 0x0000d90303530000 // Cutpoint mask for larger chunks
)

// ChunkDescriptor identifies an encrypted deduplicated slice
type ChunkDescriptor struct {
	ChunkHash     string `json:"hash"`
	OriginalSize  uint32 `json:"orig_size"`
	EncryptedSize uint32 `json:"enc_size"`
}

// BackupManifest describes the encrypted snapshot hierarchy
type BackupManifest struct {
	BackupID        uint64            `json:"backup_id"`
	CreatedAt       int64             `json:"created_at"`
	TotalPlainBytes uint64            `json:"total_plain_bytes"`
	TotalChunkBytes uint64            `json:"total_chunk_bytes"`
	Salt            []byte            `json:"salt"` // 16 bytes Argon2/HKDF salt
	Chunks          []ChunkDescriptor `json:"chunks"`
	Signature       []byte            `json:"signature"`
}

// SnapshotEngine manages FastCDC chunking, deduplication, and snapshot creation
type SnapshotEngine struct {
	mu           sync.RWMutex
	backupKey    [32]byte
	knownChunks  map[string]bool // In-memory deduplication set
	storageDir   string
}

// NewSnapshotEngine creates a backup engine with derived master key
func NewSnapshotEngine(backupKey [32]byte, storageDir string) (*SnapshotEngine, error) {
	chunksDir := filepath.Join(storageDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0700); err != nil {
		return nil, err
	}

	return &SnapshotEngine{
		backupKey:   backupKey,
		knownChunks: make(map[string]bool),
		storageDir:  storageDir,
	}, nil
}

// -----------------------------------------------------------------------------
// FASTCDC (CONTENT-DEFINED CHUNKING)
// -----------------------------------------------------------------------------

// FastCDCSplit partitions arbitrary stream data into variable-size deduplicated chunks
func FastCDCSplit(data []byte) [][]byte {
	n := len(data)
	if n <= MinChunkSize {
		return [][]byte{data}
	}

	var chunks [][]byte
	cursor := 0

	for cursor < n {
		remaining := n - cursor
		if remaining <= MinChunkSize {
			chunks = append(chunks, data[cursor:])
			break
		}

		maxBoundary := cursor + MaxChunkSize
		if maxBoundary > n {
			maxBoundary = n
		}

		splitPos := maxBoundary
		fingerprint := uint64(0)
		targetLimit := cursor + TargetChunkSize
		if targetLimit > maxBoundary {
			targetLimit = maxBoundary
		}

		// Phase 1: Small mask scanning
		i := cursor + MinChunkSize
		for ; i < targetLimit; i++ {
			fingerprint = (fingerprint << 1) + uint64(data[i])
			if (fingerprint & FastCDCMaskS) == 0 {
				splitPos = i
				break
			}
		}

		// Phase 2: Large mask scanning if not cut in Phase 1
		if splitPos == maxBoundary {
			for ; i < maxBoundary; i++ {
				fingerprint = (fingerprint << 1) + uint64(data[i])
				if (fingerprint & FastCDCMaskL) == 0 {
					splitPos = i
					break
				}
			}
		}

		chunks = append(chunks, data[cursor:splitPos])
		cursor = splitPos
	}

	return chunks
}

// -----------------------------------------------------------------------------
// COMPRESSION & ENCRYPTION
// -----------------------------------------------------------------------------

func compressChunk(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressChunk(compressed []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func (e *SnapshotEngine) encryptChunk(compressed []byte, hashHex string) ([]byte, error) {
	block, err := aes.NewCipher(e.backupKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, compressed, []byte(hashHex))
	out := make([]byte, 0, len(nonce)+len(ciphertext))
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (e *SnapshotEngine) decryptChunk(encData []byte, hashHex string) ([]byte, error) {
	if len(encData) < 12 {
		return nil, errors.New("backup: chunk too small")
	}

	block, err := aes.NewCipher(e.backupKey[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	nonce := encData[:nonceSize]
	ciphertext := encData[nonceSize:]

	compressed, err := aead.Open(nil, nonce, ciphertext, []byte(hashHex))
	if err != nil {
		return nil, err
	}

	return decompressChunk(compressed)
}

// -----------------------------------------------------------------------------
// SNAPSHOT CREATION & MANIFEST PACKING
// -----------------------------------------------------------------------------

// ProcessFileIntoChunks splits, deduplicates, and saves newly discovered chunks to local cache
func (e *SnapshotEngine) ProcessFileIntoChunks(rawBytes []byte) ([]ChunkDescriptor, uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rawChunks := FastCDCSplit(rawBytes)
	descriptors := make([]ChunkDescriptor, 0, len(rawChunks))
	var totalChunkBytes uint64

	for _, rawChunk := range rawChunks {
		h := sha256.Sum256(rawChunk)
		hashHex := hex.EncodeToString(h[:])

		compressed, err := compressChunk(rawChunk)
		if err != nil {
			return nil, 0, err
		}

		encrypted, err := e.encryptChunk(compressed, hashHex)
		if err != nil {
			return nil, 0, err
		}

		totalChunkBytes += uint64(len(encrypted))

		// Check deduplication
		prefix := hashHex[:2]
		chunkSubdir := filepath.Join(e.storageDir, "chunks", prefix)
		chunkFilePath := filepath.Join(chunkSubdir, hashHex+".bin")

		if !e.knownChunks[hashHex] {
			_ = os.MkdirAll(chunkSubdir, 0700)
			if err := os.WriteFile(chunkFilePath, encrypted, 0600); err != nil {
				return nil, 0, err
			}
			e.knownChunks[hashHex] = true
		}

		descriptors = append(descriptors, ChunkDescriptor{
			ChunkHash:     hashHex,
			OriginalSize:  uint32(len(rawChunk)),
			EncryptedSize: uint32(len(encrypted)),
		})
	}

	return descriptors, totalChunkBytes, nil
}

// CreateManifest bundles chunk descriptors into an encrypted signed root manifest
func (e *SnapshotEngine) CreateManifest(descriptors []ChunkDescriptor, totalPlainBytes, totalChunkBytes uint64) ([]byte, error) {
	manifest := BackupManifest{
		BackupID:        uint64(time.Now().UnixNano()),
		CreatedAt:       time.Now().Unix(),
		TotalPlainBytes: totalPlainBytes,
		TotalChunkBytes: totalChunkBytes,
		Chunks:          descriptors,
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	// HMAC authentication of manifest
	mac := hmac.New(sha256.New, e.backupKey[:])
	mac.Write(manifestBytes)
	manifest.Signature = mac.Sum(nil)

	signedManifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	return e.encryptChunk(signedManifestBytes, "manifest.enc")
}

// ReadChunk reads an encrypted chunk from disk by hash
func (e *SnapshotEngine) ReadChunk(hashHex string) ([]byte, error) {
	prefix := hashHex[:2]
	chunkPath := filepath.Join(e.storageDir, "chunks", prefix, hashHex+".bin")
	return os.ReadFile(chunkPath)
}
