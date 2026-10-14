// File: core/pkg/media/swarm_transfer.go
package media

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MediaChunkSize = 256 * 1024 // 256 KB per swarm chunk
)

// MediaChunkDescriptor stores verification and size metadata for an encrypted chunk
type MediaChunkDescriptor struct {
	Index         uint32 `json:"index"`
	ChunkHash     string `json:"hash"`
	OriginalSize  uint32 `json:"orig_size"`
	EncryptedSize uint32 `json:"enc_size"`
}

// MediaManifest represents the cryptographic descriptor shared in chat messages
type MediaManifest struct {
	FileID        string                 `json:"file_id"`
	FileName      string                 `json:"file_name"`
	MimeType      string                 `json:"mime_type"`
	TotalBytes    uint64                 `json:"total_bytes"`
	TotalChunks   uint32                 `json:"total_chunks"`
	RootMerkle    string                 `json:"root_merkle"`
	FileKey       []byte                 `json:"file_key"`   // 32 bytes AEAD key
	NoncePrefix   []byte                 `json:"nonce_prefix"` // 8 bytes nonce prefix
	BlurHash      string                 `json:"blur_hash"`   // Instant compact thumbnail
	Chunks        []MediaChunkDescriptor `json:"chunks"`
}

// SwarmManager coordinates multi-threaded chunk downloading, seeding, and peer distribution
type SwarmManager struct {
	mu           sync.RWMutex
	storageDir   string
	listenPort   int
	localSeeding map[string]map[uint32]bool // FileID -> set of owned chunk indices
	activePeers  map[string][]string        // FileID -> list of peer IPv6 addresses
	httpServer   *http.Server
}

// NewSwarmManager initializes a local media cache and peer seeding endpoint
func NewSwarmManager(storageDir string, listenPort int) (*SwarmManager, error) {
	if listenPort <= 0 {
		listenPort = 54332
	}
	mediaDir := filepath.Join(storageDir, "media_cache")
	if err := os.MkdirAll(mediaDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create media storage: %w", err)
	}

	sm := &SwarmManager{
		storageDir:   mediaDir,
		listenPort:   listenPort,
		localSeeding: make(map[string]map[uint32]bool),
		activePeers:  make(map[string][]string),
	}

	sm.startSeedingServer()
	return sm, nil
}

// ComputeBlurHash creates a compact representation string for immediate image preview
func ComputeBlurHash(imgBytes []byte) string {
	if len(imgBytes) == 0 {
		return "LEHV6nWB2yk8pyo0adR*.7kCMdnj" // Default neutral blurhash
	}
	// Deterministic pseudo-blurhash encoding from image entropy
	h := sha256.Sum256(imgBytes)
	encoded := hex.EncodeToString(h[:14])
	return "L" + encoded[:26]
}

// EncryptAndCreateManifest splits an input file into 256 KB encrypted chunks and builds manifest
func (sm *SwarmManager) EncryptAndCreateManifest(
	fileName, mimeType string,
	rawReader io.Reader,
	thumbnailBytes []byte,
) (*MediaManifest, error) {
	fileKey := make([]byte, 32)
	if _, err := rand.Read(fileKey); err != nil {
		return nil, err
	}

	noncePrefix := make([]byte, 8)
	if _, err := rand.Read(noncePrefix); err != nil {
		return nil, err
	}

	fileIDHash := sha256.Sum256(append(fileKey, []byte(fileName)...))
	fileID := hex.EncodeToString(fileIDHash[:16])

	fileFolder := filepath.Join(sm.storageDir, fileID)
	if err := os.MkdirAll(fileFolder, 0700); err != nil {
		return nil, err
	}

	var descriptors []MediaChunkDescriptor
	var totalBytes uint64
	var chunkIndex uint32
	var merkleLeaves [][]byte

	block, err := aes.NewCipher(fileKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, MediaChunkSize)
	for {
		bytesRead, readErr := io.ReadFull(rawReader, buf)
		if bytesRead > 0 {
			chunkData := buf[:bytesRead]
			totalBytes += uint64(bytesRead)

			// Form 12-byte nonce: 8 bytes prefix + 4 bytes chunkIndex (BigEndian)
			nonce := make([]byte, 12)
			copy(nonce[:8], noncePrefix)
			binary.BigEndian.PutUint32(nonce[8:12], chunkIndex)

			chunkHash := sha256.Sum256(chunkData)
			hashHex := hex.EncodeToString(chunkHash[:])
			merkleLeaves = append(merkleLeaves, chunkHash[:])

			// Encrypt chunk
			ciphertext := aead.Seal(nil, nonce, chunkData, []byte(hashHex))

			// Write encrypted chunk to disk cache
			chunkPath := filepath.Join(fileFolder, fmt.Sprintf("%d.bin", chunkIndex))
			if err := os.WriteFile(chunkPath, ciphertext, 0600); err != nil {
				return nil, err
			}

			descriptors = append(descriptors, MediaChunkDescriptor{
				Index:         chunkIndex,
				ChunkHash:     hashHex,
				OriginalSize:  uint32(bytesRead),
				EncryptedSize: uint32(len(ciphertext)),
			})

			sm.mu.Lock()
			if sm.localSeeding[fileID] == nil {
				sm.localSeeding[fileID] = make(map[uint32]bool)
			}
			sm.localSeeding[fileID][chunkIndex] = true
			sm.mu.Unlock()

			chunkIndex++
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}

	rootMerkle := computeMerkleRoot(merkleLeaves)
	blurHash := ComputeBlurHash(thumbnailBytes)

	return &MediaManifest{
		FileID:      fileID,
		FileName:    fileName,
		MimeType:    mimeType,
		TotalBytes:  totalBytes,
		TotalChunks: chunkIndex,
		RootMerkle:  hex.EncodeToString(rootMerkle),
		FileKey:     fileKey,
		NoncePrefix: noncePrefix,
		BlurHash:    blurHash,
		Chunks:      descriptors,
	}, nil
}

// DownloadSwarmFile orchestrates parallel chunk downloading from peer overlay nodes
func (sm *SwarmManager) DownloadSwarmFile(
	ctx context.Context,
	manifest *MediaManifest,
	peerIPv6List []string,
	onProgress func(completedChunks, totalChunks uint32),
) ([]byte, error) {
	sm.mu.Lock()
	sm.activePeers[manifest.FileID] = peerIPv6List
	sm.mu.Unlock()

	fileFolder := filepath.Join(sm.storageDir, manifest.FileID)
	_ = os.MkdirAll(fileFolder, 0700)

	type downloadTask struct {
		chunk MediaChunkDescriptor
	}
	taskChan := make(chan downloadTask, len(manifest.Chunks))
	for _, ch := range manifest.Chunks {
		taskChan <- downloadTask{chunk: ch}
	}
	close(taskChan)

	var completed uint32
	workerCount := 4
	var wg sync.WaitGroup
	var downloadErr error
	var errOnce sync.Once

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				chunkPath := filepath.Join(fileFolder, fmt.Sprintf("%d.bin", task.chunk.Index))

				// Check if already in cache
				if _, err := os.Stat(chunkPath); err == nil {
					newDone := atomic.AddUint32(&completed, 1)
					if onProgress != nil {
						onProgress(newDone, manifest.TotalChunks)
					}
					continue
				}

				// Fetch from peer swarm
				data, err := sm.fetchChunkFromSwarm(ctx, manifest.FileID, task.chunk.Index, peerIPv6List)
				if err != nil {
					errOnce.Do(func() { downloadErr = err })
					return
				}

				if err := os.WriteFile(chunkPath, data, 0600); err != nil {
					errOnce.Do(func() { downloadErr = err })
					return
				}

				sm.mu.Lock()
				if sm.localSeeding[manifest.FileID] == nil {
					sm.localSeeding[manifest.FileID] = make(map[uint32]bool)
				}
				sm.localSeeding[manifest.FileID][task.chunk.Index] = true
				sm.mu.Unlock()

				newDone := atomic.AddUint32(&completed, 1)
				if onProgress != nil {
					onProgress(newDone, manifest.TotalChunks)
				}
			}
		}()
	}

	wg.Wait()
	if downloadErr != nil {
		return nil, downloadErr
	}

	// Assemble and decrypt full file in memory/stream
	return sm.assembleAndDecrypt(manifest)
}

func (sm *SwarmManager) fetchChunkFromSwarm(ctx context.Context, fileID string, index uint32, peers []string) ([]byte, error) {
	client := &http.Client{Timeout: 8 * time.Second}

	for _, peerIP := range peers {
		url := fmt.Sprintf("http://[%s]:%d/swarm/chunk/%s/%d", peerIP, sm.listenPort, fileID, index)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				_ = resp.Body.Close()
			}
			continue
		}

		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr == nil && len(data) > 0 {
			return data, nil
		}
	}

	return nil, fmt.Errorf("swarm: failed to fetch chunk %d from any peer", index)
}

func (sm *SwarmManager) assembleAndDecrypt(manifest *MediaManifest) ([]byte, error) {
	block, err := aes.NewCipher(manifest.FileKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	var fullBuf bytes.Buffer
	fileFolder := filepath.Join(sm.storageDir, manifest.FileID)

	for _, ch := range manifest.Chunks {
		chunkPath := filepath.Join(fileFolder, fmt.Sprintf("%d.bin", ch.Index))
		encData, err := os.ReadFile(chunkPath)
		if err != nil {
			return nil, err
		}

		nonce := make([]byte, 12)
		copy(nonce[:8], manifest.NoncePrefix)
		binary.BigEndian.PutUint32(nonce[8:12], ch.Index)

		plaintext, err := aead.Open(nil, nonce, encData, []byte(ch.ChunkHash))
		if err != nil {
			return nil, fmt.Errorf("chunk %d authentication error: %w", ch.Index, err)
		}

		// Verify SHA-256 integrity
		h := sha256.Sum256(plaintext)
		if hex.EncodeToString(h[:]) != ch.ChunkHash {
			return nil, fmt.Errorf("chunk %d hash mismatch", ch.Index)
		}

		fullBuf.Write(plaintext)
	}

	return fullBuf.Bytes(), nil
}

// Seeding HTTP server over Yggdrasil IPv6 interface
func (sm *SwarmManager) startSeedingServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/swarm/chunk/", func(w http.ResponseWriter, r *http.Request) {
		parts := filepath.SplitList(r.URL.Path)
		_ = parts
		urlParts := splitURLPath(r.URL.Path)
		if len(urlParts) < 4 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		fileID := urlParts[2]
		var chunkIdx uint32
		if _, err := fmt.Sscanf(urlParts[3], "%d", &chunkIdx); err != nil {
			http.Error(w, "invalid index", http.StatusBadRequest)
			return
		}

		chunkPath := filepath.Join(sm.storageDir, fileID, fmt.Sprintf("%d.bin", chunkIdx))
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			http.Error(w, "chunk not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(data)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", sm.listenPort),
		Handler: mux,
	}
	sm.httpServer = server

	go func() {
		_ = server.ListenAndServe()
	}()
}

func computeMerkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return make([]byte, 32)
	}
	if len(leaves) == 1 {
		return leaves[0]
	}

	var nextLevel [][]byte
	for i := 0; i < len(leaves); i += 2 {
		if i+1 < len(leaves) {
			h := sha256.Sum256(append(leaves[i], leaves[i+1]...))
			nextLevel = append(nextLevel, h[:])
		} else {
			nextLevel = append(nextLevel, leaves[i])
		}
	}
	return computeMerkleRoot(nextLevel)
}

func splitURLPath(p string) []string {
	var parts []string
	for _, s := range filepath.SplitList(p) {
		_ = s
	}
	for _, seg := range bytes.Split([]byte(p), []byte("/")) {
		if len(seg) > 0 {
			parts = append(parts, string(seg))
		}
	}
	return parts
}
