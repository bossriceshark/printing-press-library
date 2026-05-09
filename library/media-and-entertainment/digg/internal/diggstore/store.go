// Package diggstore extends the generated SQLite store with the
// Digg-AI-specific tables that hold parsed clusters, snapshots,
// authors, and events.
//
// Schema additions live here (rather than in the generated store
// package) so a regeneration of the printed CLI does not blow them
// away. EnsureSchema is idempotent and can run on every command.
package diggstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/digg/internal/diggparse"
)

// EnsureSchema creates the Digg-specific tables if they don't already
// exist. Safe to call repeatedly; uses CREATE TABLE IF NOT EXISTS.
func EnsureSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS digg_clusters (
			cluster_id TEXT PRIMARY KEY,
			cluster_url_id TEXT,
			label TEXT,
			title TEXT,
			tldr TEXT,
			url TEXT,
			permalink TEXT,
			topic TEXT,
			current_rank INTEGER,
			peak_rank INTEGER,
			previous_rank INTEGER,
			delta INTEGER,
			gravity_score REAL,
			score_components_json TEXT,
			evidence_json TEXT,
			numerator_count INTEGER,
			numerator_label TEXT,
			percent_above_average REAL,
			replacement_rationale TEXT,
			pos6h REAL, pos12h REAL, pos24h REAL, pos_last REAL,
			bookmarks INTEGER, likes INTEGER, comments INTEGER, replies INTEGER,
			quotes INTEGER, views INTEGER, view_count INTEGER, impressions INTEGER,
			retweets INTEGER, quote_tweets INTEGER,
			source_title TEXT,
			hacker_news_json TEXT,
			techmeme_json TEXT,
			external_feeds_json TEXT,
			authors_json TEXT,
			activity_at TEXT,
			computed_at TEXT,
			first_post_at TEXT,
			raw_json TEXT,
			fetched_at TEXT NOT NULL,
			last_seen_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_clusters_rank ON digg_clusters(current_rank)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_clusters_delta ON digg_clusters(delta DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_clusters_url_id ON digg_clusters(cluster_url_id)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_clusters_last_seen ON digg_clusters(last_seen_at)`,

		// Per-snapshot rank/score history. One row per (cluster, fetched_at).
		`CREATE TABLE IF NOT EXISTS digg_snapshots (
			cluster_id TEXT NOT NULL,
			fetched_at TEXT NOT NULL,
			current_rank INTEGER,
			peak_rank INTEGER,
			previous_rank INTEGER,
			delta INTEGER,
			gravity_score REAL,
			pos6h REAL, pos12h REAL, pos24h REAL,
			likes INTEGER, views INTEGER, impressions INTEGER,
			PRIMARY KEY (cluster_id, fetched_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_snapshots_at ON digg_snapshots(fetched_at)`,

		// Authors (the Digg AI 1000).
		`CREATE TABLE IF NOT EXISTS digg_authors (
			username TEXT PRIMARY KEY,
			display_name TEXT,
			x_id TEXT,
			avatar_url TEXT,
			influence REAL,
			podist REAL,
			contributed_count INTEGER DEFAULT 0,
			last_seen_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_authors_influence ON digg_authors(influence DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_authors_count ON digg_authors(contributed_count DESC)`,

		// Author membership in clusters.
		`CREATE TABLE IF NOT EXISTS digg_cluster_authors (
			cluster_id TEXT NOT NULL,
			username TEXT NOT NULL,
			post_type TEXT,
			post_x_id TEXT,
			post_permalink TEXT,
			PRIMARY KEY (cluster_id, username, post_x_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_cluster_authors_user ON digg_cluster_authors(username)`,

		// /api/trending/status events.
		`CREATE TABLE IF NOT EXISTS digg_events (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			run_id TEXT,
			cluster_id TEXT,
			label TEXT,
			username TEXT,
			post_type TEXT,
			post_x_id TEXT,
			permalink TEXT,
			delta INTEGER,
			current_rank INTEGER,
			previous_rank INTEGER,
			count INTEGER,
			total INTEGER,
			original_posts INTEGER,
			retweets INTEGER,
			quote_tweets INTEGER,
			replies INTEGER,
			links INTEGER,
			videos INTEGER,
			images INTEGER,
			embedded_count INTEGER,
			total_count INTEGER,
			at TEXT,
			created_at TEXT,
			dedupe_key TEXT,
			raw_json TEXT,
			fetched_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_events_type_at ON digg_events(type, at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_events_cluster ON digg_events(cluster_id)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_events_at ON digg_events(at DESC)`,

		// Replacement archaeology — derived during sync.
		`CREATE TABLE IF NOT EXISTS digg_replacements (
			cluster_id TEXT NOT NULL,
			observed_at TEXT NOT NULL,
			rationale TEXT,
			previous_rank INTEGER,
			cluster_url_id TEXT,
			label TEXT,
			PRIMARY KEY (cluster_id, observed_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_digg_replacements_at ON digg_replacements(observed_at DESC)`,

		// FTS5 over cluster searchable text.
		`CREATE VIRTUAL TABLE IF NOT EXISTS digg_clusters_fts USING fts5(
			cluster_id UNINDEXED,
			cluster_url_id UNINDEXED,
			label,
			title,
			tldr,
			source_title,
			tokenize='porter unicode61'
		)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("ensuring digg schema: %w (stmt: %s)", err, firstLine(q))
		}
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i > 0 {
		return s[:i]
	}
	return s
}

// UpsertCluster writes a cluster row. The first time we see a clusterId,
// fetched_at is set to now. Every write updates last_seen_at.
func UpsertCluster(db *sql.DB, c diggparse.Cluster, fetchedAt time.Time) error {
	authorsJSON, _ := json.Marshal(c.Authors)
	scJSON := raw(c.ScoreComponents)
	evJSON := raw(c.Evidence)
	hnJSON := raw(c.HackerNews)
	tmJSON := raw(c.Techmeme)
	extJSON := raw(c.ExternalFeeds)
	rawJSON := raw(c.RawJSON)
	now := fetchedAt.UTC().Format(time.RFC3339Nano)

	_, err := db.Exec(`
		INSERT INTO digg_clusters (
			cluster_id, cluster_url_id, label, title, tldr, url, permalink, topic,
			current_rank, peak_rank, previous_rank, delta, gravity_score,
			score_components_json, evidence_json,
			numerator_count, numerator_label, percent_above_average,
			replacement_rationale,
			pos6h, pos12h, pos24h, pos_last,
			bookmarks, likes, comments, replies, quotes, views, view_count, impressions,
			retweets, quote_tweets, source_title,
			hacker_news_json, techmeme_json, external_feeds_json,
			authors_json, activity_at, computed_at, first_post_at,
			raw_json, fetched_at, last_seen_at
		) VALUES (
			?,?,?,?,?,?,?,?,
			?,?,?,?,?,
			?,?,
			?,?,?,
			?,
			?,?,?,?,
			?,?,?,?,?,?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,?,
			?,?,?
		) ON CONFLICT(cluster_id) DO UPDATE SET
			cluster_url_id=COALESCE(excluded.cluster_url_id, digg_clusters.cluster_url_id),
			label=COALESCE(NULLIF(excluded.label,''), digg_clusters.label),
			title=COALESCE(NULLIF(excluded.title,''), digg_clusters.title),
			tldr=COALESCE(NULLIF(excluded.tldr,''), digg_clusters.tldr),
			url=COALESCE(NULLIF(excluded.url,''), digg_clusters.url),
			permalink=COALESCE(NULLIF(excluded.permalink,''), digg_clusters.permalink),
			topic=COALESCE(NULLIF(excluded.topic,''), digg_clusters.topic),
			current_rank=excluded.current_rank,
			peak_rank=MAX(IFNULL(digg_clusters.peak_rank, 9999), IFNULL(excluded.peak_rank, 9999)) * (CASE WHEN excluded.peak_rank IS NULL AND digg_clusters.peak_rank IS NULL THEN 0 ELSE 1 END),
			previous_rank=excluded.previous_rank,
			delta=excluded.delta,
			gravity_score=excluded.gravity_score,
			score_components_json=COALESCE(excluded.score_components_json, digg_clusters.score_components_json),
			evidence_json=COALESCE(excluded.evidence_json, digg_clusters.evidence_json),
			numerator_count=excluded.numerator_count,
			numerator_label=excluded.numerator_label,
			percent_above_average=excluded.percent_above_average,
			replacement_rationale=COALESCE(NULLIF(excluded.replacement_rationale,''), digg_clusters.replacement_rationale),
			pos6h=excluded.pos6h,
			pos12h=excluded.pos12h,
			pos24h=excluded.pos24h,
			pos_last=excluded.pos_last,
			bookmarks=excluded.bookmarks,
			likes=excluded.likes,
			comments=excluded.comments,
			replies=excluded.replies,
			quotes=excluded.quotes,
			views=excluded.views,
			view_count=excluded.view_count,
			impressions=excluded.impressions,
			retweets=excluded.retweets,
			quote_tweets=excluded.quote_tweets,
			source_title=COALESCE(NULLIF(excluded.source_title,''), digg_clusters.source_title),
			hacker_news_json=COALESCE(excluded.hacker_news_json, digg_clusters.hacker_news_json),
			techmeme_json=COALESCE(excluded.techmeme_json, digg_clusters.techmeme_json),
			external_feeds_json=COALESCE(excluded.external_feeds_json, digg_clusters.external_feeds_json),
			authors_json=COALESCE(NULLIF(excluded.authors_json,'null'), digg_clusters.authors_json),
			activity_at=COALESCE(NULLIF(excluded.activity_at,''), digg_clusters.activity_at),
			computed_at=COALESCE(NULLIF(excluded.computed_at,''), digg_clusters.computed_at),
			first_post_at=COALESCE(NULLIF(excluded.first_post_at,''), digg_clusters.first_post_at),
			raw_json=excluded.raw_json,
			last_seen_at=excluded.last_seen_at
	`,
		c.ClusterID, c.ClusterURLID, c.Label, c.Title, c.TLDR, c.URL, c.Permalink, c.Topic,
		c.CurrentRank, nullableInt(c.PeakRank), c.PreviousRank, c.Delta, c.GravityScore,
		scJSON, evJSON,
		c.NumeratorCount, c.NumeratorLabel, c.PercentAboveAverage,
		c.ReplacementRationale,
		c.Pos6h, c.Pos12h, c.Pos24h, c.PosLast,
		c.Bookmarks, c.Likes, c.Comments, c.Replies, c.Quotes, c.Views, c.ViewCount, c.Impressions,
		c.Retweets, c.QuoteTweets, c.SourceTitle,
		hnJSON, tmJSON, extJSON,
		string(authorsJSON), c.ActivityAt, c.ComputedAt, c.FirstPostAt,
		rawJSON, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert cluster %s: %w", c.ClusterID, err)
	}

	// Snapshot row.
	if _, err := db.Exec(`
		INSERT OR REPLACE INTO digg_snapshots (
			cluster_id, fetched_at, current_rank, peak_rank, previous_rank, delta,
			gravity_score, pos6h, pos12h, pos24h, likes, views, impressions
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
	`, c.ClusterID, now, c.CurrentRank, nullableInt(c.PeakRank), c.PreviousRank, c.Delta,
		c.GravityScore, c.Pos6h, c.Pos12h, c.Pos24h, c.Likes, c.Views, c.Impressions); err != nil {
		return fmt.Errorf("snapshot %s: %w", c.ClusterID, err)
	}

	// Authors and membership.
	for _, a := range c.Authors {
		if a.Username == "" {
			continue
		}
		if err := upsertAuthor(db, a, now); err != nil {
			return err
		}
		if _, err := db.Exec(`
			INSERT OR REPLACE INTO digg_cluster_authors
			(cluster_id, username, post_type, post_x_id, post_permalink)
			VALUES (?,?,?,?,?)
		`, c.ClusterID, a.Username, a.PostType, a.PostXID, a.PostPermalink); err != nil {
			return fmt.Errorf("cluster_author %s/%s: %w", c.ClusterID, a.Username, err)
		}
	}

	// FTS row.
	if _, err := db.Exec(`DELETE FROM digg_clusters_fts WHERE cluster_id = ?`, c.ClusterID); err != nil {
		return fmt.Errorf("fts delete: %w", err)
	}
	if _, err := db.Exec(`
		INSERT INTO digg_clusters_fts (cluster_id, cluster_url_id, label, title, tldr, source_title)
		VALUES (?,?,?,?,?,?)
	`, c.ClusterID, c.ClusterURLID, c.Label, c.Title, c.TLDR, c.SourceTitle); err != nil {
		return fmt.Errorf("fts insert: %w", err)
	}

	return nil
}

func upsertAuthor(db *sql.DB, a diggparse.ClusterAuthor, now string) error {
	_, err := db.Exec(`
		INSERT INTO digg_authors (username, display_name, x_id, avatar_url, influence, podist, contributed_count, last_seen_at)
		VALUES (?,?,?,?,?,?,1,?)
		ON CONFLICT(username) DO UPDATE SET
			display_name=COALESCE(NULLIF(excluded.display_name,''), digg_authors.display_name),
			x_id=COALESCE(NULLIF(excluded.x_id,''), digg_authors.x_id),
			avatar_url=COALESCE(NULLIF(excluded.avatar_url,''), digg_authors.avatar_url),
			influence=CASE WHEN excluded.influence > 0 THEN excluded.influence ELSE digg_authors.influence END,
			podist=CASE WHEN excluded.podist > 0 THEN excluded.podist ELSE digg_authors.podist END,
			contributed_count=digg_authors.contributed_count + 1,
			last_seen_at=excluded.last_seen_at
	`, a.Username, a.DisplayName, a.XID, a.AvatarURL, a.Influence, a.Podist, now)
	return err
}

// UpsertEvent writes one event row from /api/trending/status.events[].
func UpsertEvent(db *sql.DB, e diggparse.Event, fetchedAt time.Time) error {
	if e.ID == "" {
		return nil
	}
	rawJSON := raw(e.RawJSON)
	now := fetchedAt.UTC().Format(time.RFC3339Nano)
	_, err := db.Exec(`
		INSERT INTO digg_events (
			id, type, run_id, cluster_id, label, username, post_type, post_x_id, permalink,
			delta, current_rank, previous_rank, count, total,
			original_posts, retweets, quote_tweets, replies, links, videos, images,
			embedded_count, total_count, at, created_at, dedupe_key, raw_json, fetched_at
		) VALUES (?,?,?,?,?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?,?,?, ?,?,?,?,?,?,?)
		ON CONFLICT(id) DO NOTHING
	`,
		e.ID, e.Type, e.RunID, e.ClusterID, e.Label, e.Username, e.PostType, e.PostXID, e.Permalink,
		e.Delta, e.CurrentRank, e.PreviousRank, e.Count, e.Total,
		e.OriginalPosts, e.Retweets, e.QuoteTweets, e.Replies, e.Links, e.Videos, e.Images,
		e.EmbeddedCount, e.TotalCount, e.At, e.CreatedAt, e.DedupeKey, rawJSON, now,
	)
	if err != nil {
		return fmt.Errorf("upsert event %s: %w", e.ID, err)
	}
	return nil
}

// RecordReplacements compares the cluster IDs we just observed against
// the cluster IDs we had in the local store and records a replacement
// row for any cluster present last sync but missing from the current
// snapshot. The "rationale" is best-effort — Digg only sometimes ships
// it; otherwise we record a synthetic "fell out of feed" rationale.
func RecordReplacements(db *sql.DB, observedClusterIDs map[string]bool, observedAt time.Time) error {
	now := observedAt.UTC().Format(time.RFC3339Nano)
	rows, err := db.Query(`
		SELECT cluster_id, cluster_url_id, label, current_rank, replacement_rationale, last_seen_at
		FROM digg_clusters
		WHERE last_seen_at < ?
		  AND last_seen_at >= datetime(?, '-2 hours')
	`, now, now)
	if err != nil {
		return fmt.Errorf("scanning replacements: %w", err)
	}
	defer rows.Close()

	type pending struct {
		id, urlID, label, rationale string
		rank                        int
	}
	var pendings []pending
	for rows.Next() {
		var id, urlID, label, rationale string
		var rank sql.NullInt64
		var lastSeen string
		if err := rows.Scan(&id, &urlID, &label, &rank, &rationale, &lastSeen); err != nil {
			return err
		}
		if observedClusterIDs[id] {
			continue
		}
		r := rationale
		if r == "" {
			r = "fell out of feed (no rationale published)"
		}
		pendings = append(pendings, pending{id: id, urlID: urlID, label: label, rationale: r, rank: int(rank.Int64)})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range pendings {
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO digg_replacements (cluster_id, observed_at, rationale, previous_rank, cluster_url_id, label)
			VALUES (?,?,?,?,?,?)
		`, p.id, now, p.rationale, p.rank, p.urlID, p.label); err != nil {
			return fmt.Errorf("record replacement %s: %w", p.id, err)
		}
	}
	return nil
}

func raw(r json.RawMessage) any {
	if len(r) == 0 {
		return nil
	}
	return string(r)
}

func nullableInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}
