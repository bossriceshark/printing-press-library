// Copyright 2026 Matt Rice and contributors. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newExportCmd(flags *rootFlags) *cobra.Command {
	var format, out string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Dump the local SQLite mirror for agent ingest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "json" && format != "csv" {
				return codeErr(2, fmt.Errorf("--format must be json or csv"))
			}
			cfg, err := flags.loadConfig()
			if err != nil {
				return err
			}
			st, err := flags.openStore(cfg)
			if err != nil {
				return err
			}
			defer st.Close()
			tables, err := st.ExportTables()
			if err != nil {
				return err
			}
			payload := map[string]any{
				"exported_at": time.Now().UTC().Format(time.RFC3339),
				"db":          cfg.DBPath,
				"source":      "local",
				"tables":      tables,
			}
			if format == "json" {
				return writeJSONExport(cmd, out, payload)
			}
			return writeCSVExport(out, tables)
		},
	}
	cmd.Flags().StringVar(&format, "format", "json", "json or csv")
	cmd.Flags().StringVar(&out, "out", "", "Output file for json, or directory for csv")
	return cmd
}

func writeJSONExport(cmd *cobra.Command, out string, payload map[string]any) error {
	var enc *json.Encoder
	if out == "" {
		enc = json.NewEncoder(cmd.OutOrStdout())
	} else {
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil && filepath.Dir(out) != "." {
			return err
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		enc = json.NewEncoder(f)
	}
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func writeCSVExport(out string, tables map[string]any) error {
	if out == "" {
		return codeErr(2, fmt.Errorf("csv export requires --out directory"))
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		return err
	}
	for name, raw := range tables {
		rows, _ := raw.([]map[string]any)
		f, err := os.Create(filepath.Join(out, name+".csv"))
		if err != nil {
			return err
		}
		w := csv.NewWriter(f)
		if len(rows) == 0 {
			_ = w.Write([]string{})
			w.Flush()
			_ = f.Close()
			continue
		}
		header := make([]string, 0, len(rows[0]))
		for k := range rows[0] {
			header = append(header, k)
		}
		if err := w.Write(header); err != nil {
			_ = f.Close()
			return err
		}
		for _, row := range rows {
			rec := make([]string, len(header))
			for i, k := range header {
				rec[i] = fmt.Sprint(row[k])
			}
			if err := w.Write(rec); err != nil {
				_ = f.Close()
				return err
			}
		}
		w.Flush()
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
