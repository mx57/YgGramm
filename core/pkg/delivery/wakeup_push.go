// File: core/pkg/delivery/wakeup_push.go
//
// pkg/delivery — каркас без вызывающего: не импортируется (см.
// TestUnreachablePackagesMatchRegistry), тестов нет, и сквозного сценария
// «сообщение разбудило устройство» не проверялось. Android-приёмник
// PushWakeupReceiver берёт wakelock и поднимает фоновый сервис, но ядро не будит:
// путь к сети у сервиса общий, а не через пуш. Обезличенного шлюза в коде нет —
// пост идёт на http://[…]:54330/api/v1/blind_wake, которого в репозитории не
// реализовано. С Шага 6 S&F-релея здесь больше нет: sf_relay.go удалён, потому что
// врал, а настоящий слепой store-and-forward живёт в pkg/rpc/sf.go и ходит через
// оверлей.
package delivery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// WakeupProvider defines the mechanism used to wake a sleeping Android device
type WakeupProvider int

const (
	ProviderUnifiedPush WakeupProvider = iota
	ProviderAnonymizedFCM
	ProviderAdaptivePolling
)

// WakeupConfig holds registration and endpoint tokens for push gateways
type WakeupConfig struct {
	ActiveProvider      WakeupProvider
	UnifiedPushEndpoint string // e.g. "https://ntfy.sh/ygggram-blind-xyz" or self-hosted ntfy/gotify
	FCMProxyGatewayIPv6 string // Yggdrasil IPv6 of anonymized FCM gateway
	FCMPushToken        string // Blind token from client
	MinKeepaliveSeconds int    // e.g. 20s when active
	MaxKeepaliveSeconds int    // e.g. 600s in deep Doze mode
}

// WakeupManager coordinates cross-provider zero-payload wakeups and polling
type WakeupManager struct {
	mu              sync.RWMutex
	cfg             WakeupConfig
	httpClient      *http.Client
	isScreenOn      bool
	isPowerSaver    bool
	currentInterval time.Duration
	stopPollChan    chan struct{}
}

// NewWakeupManager creates a push wakeup coordinator
func NewWakeupManager(cfg WakeupConfig) *WakeupManager {
	if cfg.MinKeepaliveSeconds <= 0 {
		cfg.MinKeepaliveSeconds = 25
	}
	if cfg.MaxKeepaliveSeconds <= 0 {
		cfg.MaxKeepaliveSeconds = 600
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableKeepAlives: false,
		MaxIdleConns:      5,
		IdleConnTimeout:   60 * time.Second,
	}

	return &WakeupManager{
		cfg:             cfg,
		httpClient:      &http.Client{Transport: transport, Timeout: 8 * time.Second},
		isScreenOn:      true,
		currentInterval: time.Duration(cfg.MinKeepaliveSeconds) * time.Second,
		stopPollChan:    make(chan struct{}),
	}
}

// -----------------------------------------------------------------------------
// 1. UNIFIEDPUSH ZERO-PAYLOAD PUSH
// -----------------------------------------------------------------------------

// SendUnifiedPushWakeup sends an empty/blind HTTP POST to the recipient's ntfy endpoint
func (m *WakeupManager) SendUnifiedPushWakeup(ctx context.Context, endpoint string) error {
	if endpoint == "" {
		return errors.New("wakeup_push: unifiedpush endpoint is empty")
	}

	// Zero-payload notification (RFC UnifiedPush specification)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte("1")))
	if err != nil {
		return err
	}
	req.Header.Set("Title", "YggGram")
	req.Header.Set("Priority", "urgent")
	req.Header.Set("Tags", "envelope")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unifiedpush post failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unifiedpush returned status: %d", resp.StatusCode)
	}

	return nil
}

// -----------------------------------------------------------------------------
// 2. ANONYMIZED FCM / HMS PROXY GATEWAY INSIDE YGGDRASIL
// -----------------------------------------------------------------------------

type fcmProxyRequest struct {
	PushToken string `json:"token"`
	Priority  string `json:"priority"`
}

// SendAnonymizedFCMWakeup routes push trigger through an internal Yggdrasil proxy gateway
// The gateway forwards the blind trigger to FCM without Google learning sender's physical IP
func (m *WakeupManager) SendAnonymizedFCMWakeup(ctx context.Context, gatewayIPv6, pushToken string) error {
	if gatewayIPv6 == "" || pushToken == "" {
		return errors.New("wakeup_push: missing gateway or push token")
	}

	proxyURL := fmt.Sprintf("http://[%s]:54330/api/v1/blind_wake", gatewayIPv6)

	payloadBytes, err := json.Marshal(fcmProxyRequest{
		PushToken: pushToken,
		Priority:  "high",
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, proxyURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("anonymized fcm gateway post error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("fcm gateway rejected request (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// -----------------------------------------------------------------------------
// 3. ADAPTIVE BATTERY-SAVING QUIC BACKGROUND POLLING
// -----------------------------------------------------------------------------

// UpdateDevicePowerState adjusts the polling interval based on Android Doze / Power Saver
func (m *WakeupManager) UpdateDevicePowerState(screenOn, powerSaver bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.isScreenOn = screenOn
	m.isPowerSaver = powerSaver

	if screenOn && !powerSaver {
		m.currentInterval = time.Duration(m.cfg.MinKeepaliveSeconds) * time.Second
	} else if powerSaver {
		m.currentInterval = time.Duration(m.cfg.MaxKeepaliveSeconds) * time.Second
	} else {
		// Screen off, normal battery: 90 seconds
		m.currentInterval = 90 * time.Second
	}
}

// StartAdaptivePollingLoop polls S&F relays at dynamic intervals
func (m *WakeupManager) StartAdaptivePollingLoop(pollCallback func() error) {
	go func() {
		for {
			m.mu.RLock()
			interval := m.currentInterval
			m.mu.RUnlock()

			select {
			case <-m.stopPollChan:
				return
			case <-time.After(interval):
				if pollCallback != nil {
					_ = pollCallback()
				}
			}
		}
	}()
}

// Stop terminates active polling loops
func (m *WakeupManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case <-m.stopPollChan:
	default:
		close(m.stopPollChan)
	}
}
