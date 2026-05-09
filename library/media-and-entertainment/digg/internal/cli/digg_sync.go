package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/digg/internal/diggparse"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/digg/internal/diggstore"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/digg/internal/store"

	"github.com/spf13/cobra"
)

// newDiggSyncCmd replaces the generated sync command. The generator
// emits a sync that walks the spec's REST resources, but Digg's data
// only flows through HTML scrape (/ai) and one JSON endpoint
// (/api/trending/status). This implementation does what the data shape
// actually requires: fetch the HTML, decode the embedded RSC stream,
// extract clusters and authors, persist them, and pull the trending
// status events on the side.
func newDiggSyncCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var withDetails bool
	var skipEvents bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync the /ai feed and /api/trending/status events into the local store",
		Long: `Pull the current Digg AI feed and the trending pipeline event stream into a local
SQLite database. The /ai page is fetched once; the embedded RSC stream is decoded
and every cluster, author, and snapshot is persisted. The /api/trending/status
endpoint is then read for pipeline events. Replacement archaeology runs at the
end to record clusters that were present in the previous sync but are absent now.

Sync is read-only against Digg. It never mutates anything upstream.`,
		Example: `  # Sync the current /ai feed and trending events
  digg-pp-cli sync

  # Skip events (only feed)
  digg-pp-cli sync --no-events

  # Also fetch each cluster's detail page (slower; populates fuller fields)
  digg-pp-cli sync --with-details`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if dbPath == "" {
				dbPath = defaultDBPath("digg-pp-cli")
			}

			s, err := store.OpenWithContext(ctx, dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer s.Close()

			db := s.DB()
			if err := diggstore.EnsureSchema(db); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			now := time.Now().UTC()

			// 1. Fetch /ai HTML
			fmt.Fprintln(out, "fetching /ai ...")
			html, err := fetchURL(ctx, "https://di.gg/ai")
			if err != nil {
				return fmt.Errorf("fetching /ai: %w", err)
			}
			clusters, embeddedEvents, _, err := diggparse.ParseHomeFeed(html)
			if err != nil {
				return fmt.Errorf("parsing /ai: %w", err)
			}
			fmt.Fprintf(out, "parsed %d clusters from /ai (%d KB)\n", len(clusters), len(html)/1024)

			// 2. Persist
			observed := make(map[string]bool, len(clusters))
			for _, c := range clusters {
				observed[c.ClusterID] = true
				if err := diggstore.UpsertCluster(db, c, now); err != nil {
					return err
				}
			}

			// Persist embedded events too (cluster_detected, fast_climb seen in /ai stream)
			for _, e := range embeddedEvents {
				if err := diggstore.UpsertEvent(db, e, now); err != nil {
					return err
				}
			}

			// 3. Replacements
			if err := diggstore.RecordReplacements(db, observed, now); err != nil {
				return err
			}

			// 4. Trending status events
			if !skipEvents {
				fmt.Fprintln(out, "fetching /api/trending/status ...")
				body, err := fetchURL(ctx, "https://di.gg/api/trending/status")
				if err != nil {
					return fmt.Errorf("fetching trending status: %w", err)
				}
				ts, err := diggparse.ParseTrendingStatus(body)
				if err != nil {
					return err
				}
				for _, e := range ts.Events {
					if err := diggstore.UpsertEvent(db, e, now); err != nil {
						return err
					}
				}
				fmt.Fprintf(out, "stored %d events; storiesToday=%d clustersToday=%d\n",
					len(ts.Events), ts.StoriesToday, ts.ClustersToday)
			}

			// 5. Optionally fetch detail pages for clusters
			if withDetails {
				fetched := 0
				for _, c := range clusters {
					if c.ClusterURLID == "" {
						continue
					}
					url := "https://di.gg/ai/" + c.ClusterURLID
					body, err := fetchURL(ctx, url)
					if err != nil {
						fmt.Fprintf(out, "  detail %s: %v\n", c.ClusterURLID, err)
						continue
					}
					more, _, _, err := diggparse.ParseHomeFeed(body)
					if err != nil || len(more) == 0 {
						continue
					}
					// Find the matching cluster object in the detail page (richer fields)
					for _, mc := range more {
						if mc.ClusterID == c.ClusterID {
							_ = diggstore.UpsertCluster(db, mc, now)
							break
						}
					}
					fetched++
					time.Sleep(500 * time.Millisecond)
				}
				fmt.Fprintf(out, "fetched %d detail pages\n", fetched)
			}

			summary := map[string]any{
				"event":           "sync_summary",
				"clusters_synced": len(clusters),
				"events_synced":   len(embeddedEvents),
				"with_details":    withDetails,
				"skip_events":     skipEvents,
				"db_path":         dbPath,
				"computed_at":     now.Format(time.RFC3339Nano),
			}
			if flags.asJSON {
				return printJSONFiltered(out, summary, flags)
			}
			fmt.Fprintf(out, "synced %d clusters into %s\n", len(clusters), dbPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/digg-pp-cli/data.db)")
	cmd.Flags().BoolVar(&withDetails, "with-details", false, "Also fetch each cluster's detail page (slower; richer fields)")
	cmd.Flags().BoolVar(&skipEvents, "no-events", false, "Skip /api/trending/status events fetch")

	cmd.Annotations = map[string]string{}
	return cmd
}

// fetchURL is a tiny stdlib HTTP client used by the digg sync and live
// commands. Identifies itself with a clear User-Agent so Digg ops can
// rate-limit it cleanly. 30s timeout; respects ctx cancellation.
func fetchURL(ctx context.Context, url string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", diggUserAgent())
	req.Header.Set("Accept", "text/html,application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func diggUserAgent() string {
	return "digg-pp-cli/0.1.0 (+https://github.com/mvanhorn/printing-press-library)"
}

// asJSONString takes any nullable JSON value and returns either a parsed
// any or nil — used by command output paths that pass raw_json straight
// through.
func asJSONString(s string) any {
	if s == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}
	return s
}

// joinNonEmpty filters out empty strings then joins.
func joinNonEmpty(xs []string, sep string) string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return strings.Join(out, sep)
}
