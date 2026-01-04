package repository

import (
	"testing"
)

func TestNewSettingsRepository(t *testing.T) {
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewSettingsRepository(nil)
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
	if repo.db != nil {
		t.Error("Expected db to be nil")
	}
}

func TestNewProxyRepository(t *testing.T) {
	// Test with nil db (just checking constructor doesn't panic)
	repo := NewProxyRepository(nil)
	if repo == nil {
		t.Fatal("Expected repository to be created")
	}
	if repo.db != nil {
		t.Error("Expected db to be nil")
	}
}
