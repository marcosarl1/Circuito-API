package storage

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/marcosarl1/Circuito-API/internal/service"
)

type fakeUploader struct {
	mutex   sync.Mutex
	calls   int
	bodies  [][]byte
	release chan struct{}
}

func (fake *fakeUploader) Upload(_ context.Context, _, _ string, body []byte, _ string) error {
	if fake.release != nil {
		<-fake.release
	}
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.calls++
	fake.bodies = append(fake.bodies, body)
	return nil
}

type fakeSyncState struct {
	mutex sync.Mutex
	state *service.SyncState
}

func (fake *fakeSyncState) GetBucketSyncState(_ context.Context) (*service.SyncState, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	return fake.state, nil
}

func (fake *fakeSyncState) SetBucketSyncState(_ context.Context, state service.SyncState) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.state = &state
	return nil
}

func testEvents() []service.Event {
	return []service.Event{{ID: "2026090001", NomeEvento: "Corrida", Cidade: "João Pessoa", Estado: "PB"}}
}

func TestTriggerUploadsWithLegacyIDKey(t *testing.T) {
	uploader := &fakeUploader{}
	syncer := NewBucketSync("bucket", "eventos_real.json",
		func(context.Context) ([]service.Event, error) { return testEvents(), nil },
		&fakeSyncState{}, uploader)

	result, err := syncer.Trigger(context.Background())
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if result.Status != "ok" || result.EventosSynced != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if uploader.calls != 1 {
		t.Fatalf("expected 1 upload, got %d", uploader.calls)
	}

	var documents []map[string]any
	if err := json.Unmarshal(uploader.bodies[0], &documents); err != nil {
		t.Fatalf("payload is not a JSON array: %v", err)
	}
	if documents[0]["_id"] != "2026090001" {
		t.Fatalf("expected _id key, got %v", documents[0])
	}
	if _, exists := documents[0]["id"]; exists {
		t.Fatalf("payload must not contain id key: %v", documents[0])
	}
}

func TestTriggerSkipsUnchangedContent(t *testing.T) {
	uploader := &fakeUploader{}
	state := &fakeSyncState{}
	syncer := NewBucketSync("bucket", "key",
		func(context.Context) ([]service.Event, error) { return testEvents(), nil },
		state, uploader)

	if _, err := syncer.Trigger(context.Background()); err != nil {
		t.Fatalf("first: %v", err)
	}
	result, err := syncer.Trigger(context.Background())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if result.Status != "unchanged" {
		t.Fatalf("expected unchanged, got %+v", result)
	}
	if uploader.calls != 1 {
		t.Fatalf("expected upload to be skipped, calls=%d", uploader.calls)
	}
}

func TestTriggerReuploadsOnChange(t *testing.T) {
	events := testEvents()
	uploader := &fakeUploader{}
	syncer := NewBucketSync("bucket", "key",
		func(context.Context) ([]service.Event, error) { return events, nil },
		&fakeSyncState{}, uploader)

	if _, err := syncer.Trigger(context.Background()); err != nil {
		t.Fatalf("first: %v", err)
	}
	events[0].NomeEvento = "Corrida Alterada"
	result, err := syncer.Trigger(context.Background())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if result.Status != "ok" || uploader.calls != 2 {
		t.Fatalf("expected re-upload, got %+v calls=%d", result, uploader.calls)
	}
}

func TestTriggerRejectsConcurrentSync(t *testing.T) {
	uploader := &fakeUploader{release: make(chan struct{})}
	syncer := NewBucketSync("bucket", "key",
		func(context.Context) ([]service.Event, error) { return testEvents(), nil },
		&fakeSyncState{}, uploader)

	done := make(chan error, 1)
	go func() {
		_, err := syncer.Trigger(context.Background())
		done <- err
	}()

	for range 1000 {
		if syncer.InProgress() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !syncer.InProgress() {
		close(uploader.release)
		<-done
		t.Fatal("first trigger never reached the upload")
	}
	if _, err := syncer.Trigger(context.Background()); !errors.Is(err, ErrSyncInProgress) {
		t.Fatalf("expected ErrSyncInProgress, got %v", err)
	}

	close(uploader.release)
	if err := <-done; err != nil {
		t.Fatalf("first trigger: %v", err)
	}
}

func TestTriggerRequiresBucket(t *testing.T) {
	syncer := NewBucketSync("", "key",
		func(context.Context) ([]service.Event, error) { return testEvents(), nil },
		&fakeSyncState{}, &fakeUploader{})

	if _, err := syncer.Trigger(context.Background()); !errors.Is(err, ErrBucketNotConfigured) {
		t.Fatalf("expected ErrBucketNotConfigured, got %v", err)
	}
}
