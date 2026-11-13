package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

// PrintText writes a human-readable report.
func PrintText(w io.Writer, res Result) {
	for _, l := range res.Layers {
		_, _ = fmt.Fprintf(w, "layer: %s\n", l)
	}
	for _, f := range res.Findings {
		loc := f.Layer
		if f.Event != "" {
			loc += "/" + f.Event
		}
		_, _ = fmt.Fprintf(w, "[%s] %s: %s\n  fix: %s\n", f.Level, loc, f.Message, f.Fix)
	}
	errs, warns, healthy := 0, 0, 0
	for _, f := range res.Findings {
		switch f.Level {
		case "error":
			errs++
		case "warning":
			warns++
		}
	}
	for _, r := range res.Reports {
		if r.Ran && r.ExitCode == 0 && r.StdoutOK {
			healthy++
		}
	}
	_, _ = fmt.Fprintf(w, "hookdoctor: %d hooks healthy, %d errors, %d warnings\n", healthy, errs, warns)
}

// PrintJSON writes a machine-readable report.
func PrintJSON(w io.Writer, res Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}
