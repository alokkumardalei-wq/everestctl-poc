// Package output renders command results in the format the user asked for.
// Keeping this in one place means every command supports table/json/yaml
// without each implementer having to think about it.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"sigs.k8s.io/yaml"
)

type Format string

const (
	Table Format = "table"
	JSON  Format = "json"
	YAML  Format = "yaml"
)

// ParseFormat validates and normalises a user-supplied format string.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "", "table":
		return Table, nil
	case "json":
		return JSON, nil
	case "yaml", "yml":
		return YAML, nil
	default:
		return "", fmt.Errorf("unknown output format %q (want one of: table, json, yaml)", s)
	}
}

// Tabular is implemented by types that know how to render themselves as a
// table. JSON / YAML are handled generically by marshalling the value.
type Tabular interface {
	TableHeader() []string
	TableRows() [][]string
}

// Render writes v to w using the requested format.
func Render(w io.Writer, f Format, v any) error {
	switch f {
	case JSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case YAML:
		b, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	case Table:
		t, ok := v.(Tabular)
		if !ok {
			// Fall back to YAML for things that don't have a table view.
			return Render(w, YAML, v)
		}
		tw := tablewriter.NewWriter(w)
		tw.SetHeader(t.TableHeader())
		tw.SetBorder(false)
		tw.SetHeaderLine(false)
		tw.SetColumnSeparator("")
		tw.SetAutoFormatHeaders(true)
		tw.SetAutoWrapText(false)
		tw.AppendBulk(t.TableRows())
		tw.Render()
		return nil
	}
	return fmt.Errorf("unsupported format %q", f)
}
