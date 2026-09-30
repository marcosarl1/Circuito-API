package service

type SyncState struct {
	SHA256   string `bson:"sha256"`
	Eventos  int    `bson:"eventos"`
	SyncedAt string `bson:"synced_at"`
}
