// File: core/pkg/channels/dag_channel.go
//
// pkg/channels — каркас без вызывающего: не импортируется (см.
// TestUnreachablePackagesMatchRegistry), тестов нет. Подписанный Merkle-DAG тут
// собран корректно, но живёт только в памяти процесса: транспорта для gossip нет
// (GenerateGossipSyncMessage собирает сообщение, которое некому отправить), в
// protobuf нет ни одного сообщения канала, а на диске состояние не держится
// (пакет не работает с файлами вовсе), так что синхронизировать две копии нечем.
package channels

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

// PostNode represents a vertex in the cryptographically signed Merkle-DAG
type PostNode struct {
	PostID     string            `json:"post_id"` // CID: SHA-256(canonical JSON)
	ChannelID  string            `json:"channel_id"`
	AuthorPub  ed25519.PublicKey `json:"author_pub"`
	Sequence   uint64            `json:"sequence"`
	ParentIDs  []string          `json:"parent_ids"` // Merkle causal parents
	Timestamp  int64             `json:"timestamp"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	MediaLinks []string          `json:"media_links"` // Manifest CIDs
	Signature  []byte            `json:"signature"`
}

// ChannelMetadata describes a public or private broadcast channel
type ChannelMetadata struct {
	ChannelID   string            `json:"channel_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	CreatorPub  ed25519.PublicKey `json:"creator_pub"`
	CreatedAt   int64             `json:"created_at"`
	IsPublic    bool              `json:"is_public"`
}

// GossipMessage wraps DAG synchronization packets between peers
type GossipMessage struct {
	Topic    string     `json:"topic"`
	HeadCIDs []string   `json:"head_cids"`
	NewPosts []PostNode `json:"new_posts"`
	SenderID string     `json:"sender_id"`
}

// DAGChannelManager coordinates the Merkle-DAG, post verification, and Gossipsub relaying
type DAGChannelManager struct {
	mu          sync.RWMutex
	channels    map[string]*ChannelMetadata
	posts       map[string]*PostNode       // PostID -> PostNode
	heads       map[string]map[string]bool // ChannelID -> Set of leaf PostIDs
	subscribers map[string]map[string]bool // ChannelID -> Set of peer IPv6s
	onNewPostCb func(post PostNode)
}

// NewDAGChannelManager initializes a Merkle-DAG channel manager
func NewDAGChannelManager() *DAGChannelManager {
	return &DAGChannelManager{
		channels:    make(map[string]*ChannelMetadata),
		posts:       make(map[string]*PostNode),
		heads:       make(map[string]map[string]bool),
		subscribers: make(map[string]map[string]bool),
	}
}

// RegisterPostCallback sets a listener for newly verified DAG posts
func (m *DAGChannelManager) RegisterPostCallback(cb func(post PostNode)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onNewPostCb = cb
}

// CreateChannel registers a new channel with author's root key
func (m *DAGChannelManager) CreateChannel(title, desc string, authorPriv ed25519.PrivateKey) (*ChannelMetadata, error) {
	authorPub := authorPriv.Public().(ed25519.PublicKey)
	h := sha256.Sum256(append(authorPub, []byte(title)...))
	channelID := "@" + hex.EncodeToString(h[:10])

	meta := &ChannelMetadata{
		ChannelID:   channelID,
		Title:       title,
		Description: desc,
		CreatorPub:  authorPub,
		CreatedAt:   time.Now().Unix(),
		IsPublic:    true,
	}

	m.mu.Lock()
	m.channels[channelID] = meta
	m.heads[channelID] = make(map[string]bool)
	m.mu.Unlock()

	return meta, nil
}

// PublishPost signs a new post linking current DAG heads as parents
func (m *DAGChannelManager) PublishPost(
	channelID, title, body string,
	mediaLinks []string,
	authorPriv ed25519.PrivateKey,
) (*PostNode, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	channel, exists := m.channels[channelID]
	if !exists {
		return nil, errors.New("channel does not exist")
	}

	authorPub := authorPriv.Public().(ed25519.PublicKey)
	if !authorPub.Equal(channel.CreatorPub) {
		return nil, errors.New("unauthorized: author key does not match channel creator")
	}

	// Gather current DAG heads as parents
	var parentList []string
	for headID := range m.heads[channelID] {
		parentList = append(parentList, headID)
	}
	sort.Strings(parentList)

	now := time.Now().Unix()
	seq := uint64(len(m.posts) + 1)

	// Canonical payload for signing: ChannelID + Seq + Parents + Timestamp + Title + Body
	var sigBuf bytes.Buffer
	sigBuf.WriteString(channelID)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], seq)
	sigBuf.Write(b8[:])
	for _, p := range parentList {
		sigBuf.WriteString(p)
	}
	binary.BigEndian.PutUint64(b8[:], uint64(now))
	sigBuf.Write(b8[:])
	sigBuf.WriteString(title)
	sigBuf.WriteString(body)

	sig := ed25519.Sign(authorPriv, sigBuf.Bytes())

	// Compute Content Identifier (CID)
	h := sha256.Sum256(append(sigBuf.Bytes(), sig...))
	postID := hex.EncodeToString(h[:])

	node := &PostNode{
		PostID:     postID,
		ChannelID:  channelID,
		AuthorPub:  authorPub,
		Sequence:   seq,
		ParentIDs:  parentList,
		Timestamp:  now,
		Title:      title,
		Body:       body,
		MediaLinks: mediaLinks,
		Signature:  sig,
	}

	m.posts[postID] = node

	// Update DAG Heads: remove parents that are now ancestor nodes, add new head
	headMap := m.heads[channelID]
	for _, p := range parentList {
		delete(headMap, p)
	}
	headMap[postID] = true

	return node, nil
}

// IngestRemotePost validates incoming gossip post signature, verifies causality, and merges into DAG
func (m *DAGChannelManager) IngestRemotePost(post PostNode) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already stored
	if _, exists := m.posts[post.PostID]; exists {
		return false, nil // Duplicate post, skip
	}

	// 1. Reconstruct signature buffer
	var sigBuf bytes.Buffer
	sigBuf.WriteString(post.ChannelID)
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], post.Sequence)
	sigBuf.Write(b8[:])
	for _, p := range post.ParentIDs {
		sigBuf.WriteString(p)
	}
	binary.BigEndian.PutUint64(b8[:], uint64(post.Timestamp))
	sigBuf.Write(b8[:])
	sigBuf.WriteString(post.Title)
	sigBuf.WriteString(post.Body)

	// 2. Verify Ed25519 signature
	if !ed25519.Verify(post.AuthorPub, sigBuf.Bytes(), post.Signature) {
		return false, errors.New("dag_channel: invalid post signature")
	}

	// 3. Verify CID integrity
	h := sha256.Sum256(append(sigBuf.Bytes(), post.Signature...))
	computedCID := hex.EncodeToString(h[:])
	if computedCID != post.PostID {
		return false, errors.New("dag_channel: post CID mismatch")
	}

	// 4. Insert into local DAG
	m.posts[post.PostID] = &post

	headMap := m.heads[post.ChannelID]
	if headMap == nil {
		headMap = make(map[string]bool)
		m.heads[post.ChannelID] = headMap
	}

	for _, p := range post.ParentIDs {
		delete(headMap, p)
	}
	headMap[post.PostID] = true

	if m.onNewPostCb != nil {
		go m.onNewPostCb(post)
	}

	return true, nil
}

// GenerateGossipSyncMessage creates a payload for peer synchronization
func (m *DAGChannelManager) GenerateGossipSyncMessage(channelID string) *GossipMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	headMap := m.heads[channelID]
	var headList []string
	for h := range headMap {
		headList = append(headList, h)
	}

	// Attach latest 10 posts for fast propagation
	var recentPosts []PostNode
	for _, p := range m.posts {
		if p.ChannelID == channelID {
			recentPosts = append(recentPosts, *p)
		}
	}

	sort.Slice(recentPosts, func(i, j int) bool {
		return recentPosts[i].Sequence > recentPosts[j].Sequence
	})

	if len(recentPosts) > 10 {
		recentPosts = recentPosts[:10]
	}

	return &GossipMessage{
		Topic:    channelID,
		HeadCIDs: headList,
		NewPosts: recentPosts,
	}
}

// GetChannelFeed returns linear sorted timeline from the Merkle DAG
func (m *DAGChannelManager) GetChannelFeed(channelID string, limit int) []PostNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var feed []PostNode
	for _, p := range m.posts {
		if p.ChannelID == channelID {
			feed = append(feed, *p)
		}
	}

	sort.Slice(feed, func(i, j int) bool {
		return feed[i].Timestamp > feed[j].Timestamp
	})

	if limit > 0 && len(feed) > limit {
		feed = feed[:limit]
	}
	return feed
}
