package diggstore

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/digg/internal/diggparse"

	_ "modernc.org/sqlite"
)

func openTempDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := EnsureSchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestEnsureSchemaIsIdempotent(t *testing.T) {
	db := openTempDB(t)
	if err := EnsureSchema(db); err != nil {
		t.Errorf("second EnsureSchema call failed: %v", err)
	}
	// Confirm the cluster table is in place.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM digg_clusters`).Scan(&n); err != nil {
		t.Fatalf("digg_clusters not queryable: %v", err)
	}
	if n != 0 {
		t.Errorf("fresh DB should have zero clusters; got %d", n)
	}
}

func TestUpsertClusterRoundTrip(t *testing.T) {
	db := openTempDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	c := diggparse.Cluster{
		ClusterID:    "c-1",
		ClusterURLID: "abcd1234",
		Label:        "hello world",
		TLDR:         "a tldr",
		CurrentRank:  3,
		Delta:        2,
		Authors: []diggparse.ClusterAuthor{
			{Username: "alice", DisplayName: "Alice", PostType: "quote", PostXID: "x1"},
		},
	}
	if err := UpsertCluster(db, c, now); err != nil {
		t.Fatal(err)
	}
	var rank int
	var label, urlID string
	if err := db.QueryRow(`SELECT cluster_url_id, label, current_rank FROM digg_clusters WHERE cluster_id = ?`, c.ClusterID).Scan(&urlID, &label, &rank); err != nil {
		t.Fatal(err)
	}
	if urlID != "abcd1234" || label != "hello world" || rank != 3 {
		t.Errorf("cluster round-trip mismatch: urlID=%q label=%q rank=%d", urlID, label, rank)
	}
	// Author row
	var username string
	if err := db.QueryRow(`SELECT username FROM digg_authors WHERE username = ?`, "alice").Scan(&username); err != nil {
		t.Fatal(err)
	}
	if username != "alice" {
		t.Errorf("author row not written")
	}
	// Membership row
	var postType string
	if err := db.QueryRow(`SELECT post_type FROM digg_cluster_authors WHERE cluster_id = ? AND username = ?`, c.ClusterID, "alice").Scan(&postType); err != nil {
		t.Fatal(err)
	}
	if postType != "quote" {
		t.Errorf("membership row not written; got post_type=%q", postType)
	}
	// Snapshot row
	var snapRank int
	if err := db.QueryRow(`SELECT current_rank FROM digg_snapshots WHERE cluster_id = ?`, c.ClusterID).Scan(&snapRank); err != nil {
		t.Fatal(err)
	}
	if snapRank != 3 {
		t.Errorf("snapshot rank mismatch: got %d", snapRank)
	}
}

func TestUpsertEvent(t *testing.T) {
	db := openTempDB(t)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	e := diggparse.Event{
		ID:           "e-1",
		Type:         "fast_climb",
		ClusterID:    "c-1",
		Label:        "Fast Climber",
		Delta:        9,
		CurrentRank:  4,
		PreviousRank: 13,
		At:           "2026-05-09T11:00:00Z",
	}
	if err := UpsertEvent(db, e, now); err != nil {
		t.Fatal(err)
	}
	// Idempotent: second insert is a no-op.
	if err := UpsertEvent(db, e, now); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM digg_events WHERE id = ?`, e.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected exactly one event row; got %d", n)
	}
	var typ string
	var delta int
	if err := db.QueryRow(`SELECT type, COALESCE(delta,0) FROM digg_events WHERE id = ?`, e.ID).Scan(&typ, &delta); err != nil {
		t.Fatal(err)
	}
	if typ != "fast_climb" || delta != 9 {
		t.Errorf("event row mismatch: type=%q delta=%d", typ, delta)
	}
}

func TestRecordReplacementsDropsClustersNotSeen(t *testing.T) {
	db := openTempDB(t)
	old := time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)

	prev := diggparse.Cluster{ClusterID: "c-old", ClusterURLID: "oldid", Label: "Old", CurrentRank: 5}
	stay := diggparse.Cluster{ClusterID: "c-stay", ClusterURLID: "stayid", Label: "Stays", CurrentRank: 6}
	if err := UpsertCluster(db, prev, old); err != nil {
		t.Fatal(err)
	}
	if err := UpsertCluster(db, stay, old); err != nil {
		t.Fatal(err)
	}

	// At "now", we observe only c-stay. c-old should be recorded as a replacement.
	if err := UpsertCluster(db, stay, now); err != nil {
		t.Fatal(err)
	}
	observed := map[string]bool{"c-stay": true}
	if err := RecordReplacements(db, observed, now); err != nil {
		t.Fatal(err)
	}
	var rationale string
	var prevRank int
	if err := db.QueryRow(`SELECT rationale, previous_rank FROM digg_replacements WHERE cluster_id = ?`, "c-old").Scan(&rationale, &prevRank); err != nil {
		t.Fatalf("replacement row not written: %v", err)
	}
	if rationale == "" {
		t.Errorf("rationale should be populated even when upstream didn't publish one")
	}
	if prevRank != 5 {
		t.Errorf("previous_rank should be 5; got %d", prevRank)
	}
}
