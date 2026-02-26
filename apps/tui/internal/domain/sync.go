package domain

import "time"

type SyncAccount struct {
	UserID           string
	Email            string
	Username         string
	LastReconciledAt *time.Time
	UpdatedAt        time.Time
}

type SyncBookLink struct {
	LocalBookID         string
	LocalFingerprint    string
	RemoteCatalogBookID string
	RemoteLibraryBookID string
	UpdatedAt           time.Time
}
