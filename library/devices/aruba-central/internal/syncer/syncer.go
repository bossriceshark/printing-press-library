// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package syncer

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/central"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/client"
	"github.com/mvanhorn/printing-press-library/library/devices/aruba-central/internal/store"
)

// DefaultResources is the mirror that brief and export expect.
var DefaultResources = []string{"sites", "devices", "clients", "alerts", "events", "firmware", "subscriptions", "wlans", "audits"}

// Result is one resource's sync outcome.
type Result struct {
	Resource  string `json:"resource"`
	Count     int    `json:"count"`
	Pages     int    `json:"pages"`
	Requests  int    `json:"requests"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
}

// Options controls a sync run.
type Options struct {
	Only     []string
	Since    time.Duration
	Limit    int
	MaxPages int
	DryRun   bool
}

// Run pulls New Central into SQLite. Events are fetched per site inside --since.
func Run(ctx context.Context, c *client.Client, st *store.Store, opt Options) ([]Result, error) {
	if opt.Since <= 0 {
		opt.Since = 24 * time.Hour
	}
	if opt.Limit <= 0 {
		opt.Limit = 100
	}
	if opt.MaxPages <= 0 {
		opt.MaxPages = 200
	}
	names := opt.Only
	if len(names) == 0 {
		names = DefaultResources
	}
	var out []Result
	needSites := false
	for _, n := range names {
		if n == "events" {
			needSites = true
		}
	}
	haveSites := false
	for _, n := range names {
		if n == "sites" {
			haveSites = true
		}
	}
	if needSites && !haveSites {
		names = append([]string{"sites"}, names...)
	}
	for _, name := range names {
		out = append(out, syncOne(ctx, c, st, name, opt))
	}
	return out, nil
}

func syncOne(ctx context.Context, c *client.Client, st *store.Store, name string, opt Options) Result {
	res := Result{Resource: name, DryRun: opt.DryRun}
	if opt.DryRun {
		res.Requests = 1
		return res
	}
	var items []map[string]any
	var err error
	switch name {
	case "sites":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-monitoring/v1/sites-health", "offset", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceSites(items)
		}
	case "devices":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-monitoring/v1/devices", "next", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceDevices(items)
		}
	case "clients":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-monitoring/v1/clients", "next", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceClients(items)
		}
	case "alerts":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-notifications/v1/alerts", "next", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceAlerts(items)
		}
		if err == nil {
			ins, pages, trunc, ierr := c.ListPages(ctx, "network-notifications/v1/insights", "next", "items", nil, opt.Limit, opt.MaxPages)
			res.Requests += pages
			res.Pages += pages
			if trunc {
				res.Truncated = true
			}
			if ierr != nil {
				err = ierr
			} else if e := st.ReplaceInsights(ins); e != nil {
				err = e
			}
		}
	case "events":
		items, res.Requests, res.Pages, res.Truncated, err = syncEvents(ctx, c, st, opt)
		if err == nil {
			err = st.ReplaceEvents(items)
		}
	case "firmware":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-services/v1/firmware-details", "next", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceFirmware(items)
		}
	case "subscriptions":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-monitoring/v1/subscription-expiring-summary", "next", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceSubscriptions(items)
		}
	case "wlans":
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-monitoring/v1/wlans", "next", "items", nil, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceWlans(items)
		}
	case "audits":
		end := time.Now().UTC()
		start := end.Add(-opt.Since)
		q := url.Values{}
		q.Set("start-at", start.Format(time.RFC3339))
		q.Set("end-at", end.Format(time.RFC3339))
		items, res.Pages, res.Truncated, err = c.ListPages(ctx, "network-services/v1/audits", "next", "items", q, opt.Limit, opt.MaxPages)
		res.Requests = res.Pages
		if err == nil {
			err = st.ReplaceAudits(items)
		}
	default:
		err = fmt.Errorf("unknown resource %q", name)
	}
	res.Count = len(items)
	if err != nil {
		res.Error = err.Error()
	}
	return res
}

func syncEvents(ctx context.Context, c *client.Client, st *store.Store, opt Options) ([]map[string]any, int, int, bool, error) {
	sites, err := st.SiteIDs()
	if err != nil {
		return nil, 0, 0, false, err
	}
	if len(sites) == 0 {
		return nil, 0, 0, false, fmt.Errorf("events sync needs sites in the local mirror")
	}
	end := time.Now().UTC()
	start := end.Add(-opt.Since)
	var all []map[string]any
	reqs, pages := 0, 0
	trunc := false
	for _, site := range sites {
		q := url.Values{}
		q.Set("context-type", "SITE")
		q.Set("context-identifier", site[0])
		q.Set("site-id", site[0])
		q.Set("start-at", start.Format(time.RFC3339))
		q.Set("end-at", end.Format(time.RFC3339))
		items, p, t, err := c.ListPages(ctx, "network-troubleshooting/v1/events", "next", "events", q, opt.Limit, opt.MaxPages)
		reqs += p
		pages += p
		if t {
			trunc = true
		}
		if err != nil {
			return all, reqs, pages, trunc, err
		}
		for _, item := range items {
			if central.Str(item, "siteId", "site_id") == "" {
				item["siteId"] = site[0]
			}
			all = append(all, item)
		}
	}
	return all, reqs, pages, trunc, nil
}

// ParseOnly splits a comma list and checks names.
func ParseOnly(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	allowed := map[string]bool{}
	for _, n := range append(DefaultResources, "insights") {
		allowed[n] = true
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !allowed[part] {
			return nil, fmt.Errorf("unknown --only value %q (want %s)", part, strings.Join(DefaultResources, ","))
		}
		out = append(out, part)
	}
	return out, nil
}
