// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func (f *rootFlags) print(cmd *cobra.Command, v any) error {
	if f.asJSON || f.agent {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	return printHuman(cmd.OutOrStdout(), v)
}

func printHuman(w io.Writer, v any) error {
	switch t := v.(type) {
	case []map[string]any:
		if len(t) == 0 {
			fmt.Fprintln(w, "(no rows)")
			return nil
		}
		keys := []string{}
		for k := range t[0] {
			if k == "raw" {
				continue
			}
			keys = append(keys, k)
		}
		fmt.Fprintln(w, strings.Join(keys, "\t"))
		for _, row := range t {
			vals := make([]string, len(keys))
			for i, k := range keys {
				vals[i] = fmt.Sprint(row[k])
			}
			fmt.Fprintln(w, strings.Join(vals, "\t"))
		}
		return nil
	default:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}

type envelope struct {
	Meta    map[string]any `json:"meta"`
	Results any            `json:"results"`
}

func wrap(source, reason string, results any) envelope {
	if results == nil {
		results = []any{}
	}
	return envelope{Meta: map[string]any{"source": source, "reason": reason, "at": time.Now().UTC().Format(time.RFC3339)}, Results: results}
}
