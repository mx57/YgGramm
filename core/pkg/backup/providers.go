// File: core/pkg/backup/providers.go
package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CloudProvider defines the unified interface for storage backends
type CloudProvider interface {
	Name() string
	UploadChunk(ctx context.Context, hashHex string, data []byte) error
	DownloadChunk(ctx context.Context, hashHex string) ([]byte, error)
	UploadManifest(ctx context.Context, manifestData []byte) error
	DownloadManifest(ctx context.Context) ([]byte, error)
	ListExistingChunks(ctx context.Context) (map[string]bool, error)
}

// -----------------------------------------------------------------------------
// 1. GOOGLE DRIVE PROVIDER (APPDATAFOLDER + YGGDRASIL PROXY FALLBACK)
// -----------------------------------------------------------------------------

type GoogleDriveProvider struct {
	accessToken string
	httpClient  *http.Client
	baseURL     string
}

func NewGoogleDriveProvider(accessToken string, yggdrasilProxyHost string) *GoogleDriveProvider {
	transport := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
	}

	// Route through Yggdrasil exit node if direct googleapis.com is blocked in Russia
	if yggdrasilProxyHost != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "tcp", yggdrasilProxyHost)
		}
	}

	return &GoogleDriveProvider{
		accessToken: accessToken,
		httpClient:  &http.Client{Transport: transport, Timeout: 20 * time.Second},
		baseURL:     "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart",
	}
}

func (p *GoogleDriveProvider) Name() string { return "Google Drive (appDataFolder)" }

func (p *GoogleDriveProvider) UploadChunk(ctx context.Context, hashHex string, data []byte) error {
	fileName := fmt.Sprintf("chunk_%s.bin", hashHex)
	return p.uploadFile(ctx, fileName, data, []string{"appDataFolder"})
}

func (p *GoogleDriveProvider) DownloadChunk(ctx context.Context, hashHex string) ([]byte, error) {
	fileName := fmt.Sprintf("chunk_%s.bin", hashHex)
	return p.downloadFileByName(ctx, fileName)
}

func (p *GoogleDriveProvider) UploadManifest(ctx context.Context, manifestData []byte) error {
	return p.uploadFile(ctx, "manifest.enc", manifestData, []string{"appDataFolder"})
}

func (p *GoogleDriveProvider) DownloadManifest(ctx context.Context) ([]byte, error) {
	return p.downloadFileByName(ctx, "manifest.enc")
}

func (p *GoogleDriveProvider) ListExistingChunks(ctx context.Context) (map[string]bool, error) {
	url := "https://www.googleapis.com/drive/v3/files?spaces=appDataFolder&fields=files(id,name)"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Files []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	set := make(map[string]bool)
	for _, f := range result.Files {
		if strings.HasPrefix(f.Name, "chunk_") && strings.HasSuffix(f.Name, ".bin") {
			hash := strings.TrimSuffix(strings.TrimPrefix(f.Name, "chunk_"), ".bin")
			set[hash] = true
		}
	}
	return set, nil
}

func (p *GoogleDriveProvider) uploadFile(ctx context.Context, name string, content []byte, parents []string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gdrive upload failed with status %d", resp.StatusCode)
	}
	return nil
}

func (p *GoogleDriveProvider) downloadFileByName(ctx context.Context, name string) ([]byte, error) {
	searchURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=name='%s'&spaces=appDataFolder&fields=files(id)", name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Files []struct {
			ID string `json:"id"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Files) == 0 {
		return nil, errors.New("file not found in google drive")
	}

	downloadURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", result.Files[0].ID)
	downReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	downReq.Header.Set("Authorization", "Bearer "+p.accessToken)

	downResp, err := p.httpClient.Do(downReq)
	if err != nil {
		return nil, err
	}
	defer downResp.Body.Close()

	return io.ReadAll(downResp.Body)
}

// -----------------------------------------------------------------------------
// 2. YANDEX.DISK PROVIDER (DOMESTIC RUSSIAN HIGH-SPEED REST API)
// -----------------------------------------------------------------------------

type YandexDiskProvider struct {
	oauthToken string
	httpClient *http.Client
}

func NewYandexDiskProvider(oauthToken string) *YandexDiskProvider {
	return &YandexDiskProvider{
		oauthToken: oauthToken,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *YandexDiskProvider) Name() string { return "Yandex.Disk (app:/)" }

func (p *YandexDiskProvider) UploadChunk(ctx context.Context, hashHex string, data []byte) error {
	path := fmt.Sprintf("app:/chunks/%s.bin", hashHex)
	return p.uploadToPath(ctx, path, data)
}

func (p *YandexDiskProvider) DownloadChunk(ctx context.Context, hashHex string) ([]byte, error) {
	path := fmt.Sprintf("app:/chunks/%s.bin", hashHex)
	return p.downloadFromPath(ctx, path)
}

func (p *YandexDiskProvider) UploadManifest(ctx context.Context, manifestData []byte) error {
	return p.uploadToPath(ctx, "app:/manifest.enc", manifestData)
}

func (p *YandexDiskProvider) DownloadManifest(ctx context.Context) ([]byte, error) {
	return p.downloadFromPath(ctx, "app:/manifest.enc")
}

func (p *YandexDiskProvider) ListExistingChunks(ctx context.Context) (map[string]bool, error) {
	apiURL := "https://cloud-api.yandex.net/v1/disk/resources?path=app:/chunks&limit=10000"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+p.oauthToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Embedded struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		} `json:"_embedded"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	set := make(map[string]bool)
	for _, item := range result.Embedded.Items {
		if strings.HasSuffix(item.Name, ".bin") {
			set[strings.TrimSuffix(item.Name, ".bin")] = true
		}
	}
	return set, nil
}

func (p *YandexDiskProvider) uploadToPath(ctx context.Context, targetPath string, data []byte) error {
	// 1. Request Upload URL
	uploadURLReq := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources/upload?path=%s&overwrite=true", targetPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uploadURLReq, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "OAuth "+p.oauthToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var uploadMeta struct {
		Href string `json:"href"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadMeta); err != nil {
		return err
	}

	// 2. PUT binary bytes to obtained Href
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadMeta.Href, bytes.NewReader(data))
	if err != nil {
		return err
	}
	putReq.Header.Set("Content-Type", "application/octet-stream")

	putResp, err := p.httpClient.Do(putReq)
	if err != nil {
		return err
	}
	defer putResp.Body.Close()

	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return fmt.Errorf("yandex disk put returned %d", putResp.StatusCode)
	}
	return nil
}

func (p *YandexDiskProvider) downloadFromPath(ctx context.Context, targetPath string) ([]byte, error) {
	downURLReq := fmt.Sprintf("https://cloud-api.yandex.net/v1/disk/resources/download?path=%s", targetPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downURLReq, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+p.oauthToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var downMeta struct {
		Href string `json:"href"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&downMeta); err != nil {
		return nil, err
	}

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downMeta.Href, nil)
	if err != nil {
		return nil, err
	}

	getResp, err := p.httpClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()

	return io.ReadAll(getResp.Body)
}

// -----------------------------------------------------------------------------
// 3. WEBDAV PROVIDER (MAIL.RU CLOUD & NEXTCLOUD)
// -----------------------------------------------------------------------------

type WebDAVProvider struct {
	endpoint string
	username string
	password string
	client   *http.Client
}

func NewWebDAVProvider(endpoint, username, password string) *WebDAVProvider {
	return &WebDAVProvider{
		endpoint: strings.TrimSuffix(endpoint, "/"),
		username: username,
		password: password,
		client:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (p *WebDAVProvider) Name() string { return "WebDAV (Mail.ru / Nextcloud)" }

func (p *WebDAVProvider) UploadChunk(ctx context.Context, hashHex string, data []byte) error {
	url := fmt.Sprintf("%s/chunks/%s.bin", p.endpoint, hashHex)
	return p.put(ctx, url, data)
}

func (p *WebDAVProvider) DownloadChunk(ctx context.Context, hashHex string) ([]byte, error) {
	url := fmt.Sprintf("%s/chunks/%s.bin", p.endpoint, hashHex)
	return p.get(ctx, url)
}

func (p *WebDAVProvider) UploadManifest(ctx context.Context, manifestData []byte) error {
	url := fmt.Sprintf("%s/manifest.enc", p.endpoint)
	return p.put(ctx, url, manifestData)
}

func (p *WebDAVProvider) DownloadManifest(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("%s/manifest.enc", p.endpoint)
	return p.get(ctx, url)
}

func (p *WebDAVProvider) ListExistingChunks(ctx context.Context) (map[string]bool, error) {
	// PROPFIND request for chunks folder
	return make(map[string]bool), nil
}

func (p *WebDAVProvider) put(ctx context.Context, url string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.username, p.password)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webdav put status: %d", resp.StatusCode)
	}
	return nil
}

func (p *WebDAVProvider) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(p.username, p.password)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// -----------------------------------------------------------------------------
// 4. LAN & YGGDRASIL DIRECT DEVICE SYNC (ANDROID <-> WINDOWS PC)
// -----------------------------------------------------------------------------

type LanDeviceSync struct {
	remoteAddress string // e.g. "192.168.1.100:54331" or Yggdrasil "[0200::...]:54331"
	client        *http.Client
}

func NewLanDeviceSync(remoteAddress string) *LanDeviceSync {
	return &LanDeviceSync{
		remoteAddress: remoteAddress,
		client:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *LanDeviceSync) Name() string { return "P2P Device Sync (LAN / Yggdrasil)" }

func (p *LanDeviceSync) UploadChunk(ctx context.Context, hashHex string, data []byte) error {
	url := fmt.Sprintf("http://%s/sync/chunk/%s", p.remoteAddress, hashHex)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (p *LanDeviceSync) DownloadChunk(ctx context.Context, hashHex string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/sync/chunk/%s", p.remoteAddress, hashHex)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (p *LanDeviceSync) UploadManifest(ctx context.Context, manifestData []byte) error {
	url := fmt.Sprintf("http://%s/sync/manifest", p.remoteAddress)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(manifestData))
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (p *LanDeviceSync) DownloadManifest(ctx context.Context) ([]byte, error) {
	url := fmt.Sprintf("http://%s/sync/manifest", p.remoteAddress)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (p *LanDeviceSync) ListExistingChunks(ctx context.Context) (map[string]bool, error) {
	url := fmt.Sprintf("http://%s/sync/chunks/list", p.remoteAddress)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chunks []string
	if err := json.NewDecoder(resp.Body).Decode(&chunks); err != nil {
		return nil, err
	}

	set := make(map[string]bool, len(chunks))
	for _, c := range chunks {
		set[c] = true
	}
	return set, nil
}
