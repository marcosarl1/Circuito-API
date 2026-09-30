package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
)

var (
	ErrBucketNotConfigured = errors.New("bucket not configured")
	ErrSyncInProgress      = errors.New("sync already in progress")
)

type SyncResult struct {
	Status        string `json:"status"`
	EventosSynced int    `json:"eventos_synced"`
}

type EventProvider func(requestContext context.Context) ([]service.Event, error)

type SyncStateStore interface {
	GetBucketSyncState(requestContext context.Context) (*service.SyncState, error)
	SetBucketSyncState(requestContext context.Context, state service.SyncState) error
}

type Uploader interface {
	Upload(requestContext context.Context, bucket, key string, body []byte, contentType string) error
}

type BucketSync struct {
	bucket   string
	key      string
	events   EventProvider
	state    SyncStateStore
	uploader Uploader
	mutex    sync.Mutex
	running  atomic.Bool
}

func NewBucketSync(bucket, key string, events EventProvider, state SyncStateStore, uploader Uploader) *BucketSync {
	return &BucketSync{bucket: bucket, key: key, events: events, state: state, uploader: uploader}
}

func (syncer *BucketSync) InProgress() bool {
	return syncer.running.Load()
}

func (syncer *BucketSync) Trigger(requestContext context.Context) (SyncResult, error) {
	if syncer.bucket == "" {
		return SyncResult{}, ErrBucketNotConfigured
	}

	syncer.mutex.Lock()
	if syncer.running.Load() {
		syncer.mutex.Unlock()
		return SyncResult{}, ErrSyncInProgress
	}
	syncer.running.Store(true)
	syncer.mutex.Unlock()
	defer syncer.running.Store(false)

	requestContext, cancel := context.WithTimeout(requestContext, 60*time.Second)
	defer cancel()

	eventos, err := syncer.events(requestContext)
	if err != nil {
		return SyncResult{}, err
	}

	payload, err := buildBucketPayload(eventos)
	if err != nil {
		return SyncResult{}, err
	}

	hash := sha256.Sum256(payload)
	fingerprint := hex.EncodeToString(hash[:])

	previous, err := syncer.state.GetBucketSyncState(requestContext)
	if err != nil {
		return SyncResult{}, err
	}
	if previous != nil && previous.SHA256 == fingerprint {
		return SyncResult{Status: "unchanged", EventosSynced: len(eventos)}, nil
	}

	if err := syncer.uploader.Upload(requestContext, syncer.bucket, syncer.key, payload, "application/json"); err != nil {
		return SyncResult{}, err
	}

	err = syncer.state.SetBucketSyncState(requestContext, service.SyncState{
		SHA256:   fingerprint,
		Eventos:  len(eventos),
		SyncedAt: service.NowISO(),
	})
	if err != nil {
		return SyncResult{}, err
	}

	return SyncResult{Status: "ok", EventosSynced: len(eventos)}, nil
}

func buildBucketPayload(eventos []service.Event) ([]byte, error) {
	documents := make([]map[string]any, 0, len(eventos))
	for _, evento := range eventos {
		raw, err := json.Marshal(evento)
		if err != nil {
			continue
		}
		var document map[string]any
		if err := json.Unmarshal(raw, &document); err != nil {
			continue
		}
		document["_id"] = document["id"]
		delete(document, "id")
		documents = append(documents, document)
	}
	return json.Marshal(documents)
}
