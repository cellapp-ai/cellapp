package appdata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

type Manifest struct {
	Version          int     `json:"version"`
	AllowDestructive bool    `json:"allowDestructive,omitempty"`
	Tables           []Table `json:"tables"`
}

type Table struct {
	Name       string   `json:"name"`
	Columns    []Column `json:"columns"`
	PrimaryKey []string `json:"primaryKey"`
	Indexes    []Index  `json:"indexes,omitempty"`
	Policies   Policies `json:"policies"`
}

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable,omitempty"`
}

type Index struct {
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique,omitempty"`
}

type Policies struct {
	Select      string `json:"select,omitempty"`
	Insert      string `json:"insert,omitempty"`
	Update      string `json:"update,omitempty"`
	Delete      string `json:"delete,omitempty"`
	OwnerColumn string `json:"ownerColumn,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
var systemColumns = map[string]bool{"ctid": true, "tableoid": true, "xmin": true, "cmin": true, "xmax": true, "cmax": true}
var columnTypes = map[string]bool{"text": true, "uuid": true, "bigint": true, "boolean": true, "timestamptz": true, "jsonb": true}

func validMode(mode string) bool {
	return mode == "" || mode == "none" || mode == "all-admitted-users" || mode == "own-rows"
}

func (m Manifest) Validate() error {
	if m.Version < 1 || m.Version > 1_000_000 || len(m.Tables) > 20 {
		return fmt.Errorf("invalid manifest version or table count")
	}
	seenTables := map[string]bool{}
	for _, table := range m.Tables {
		if table.Name == "rpc" {
			return fmt.Errorf("reserved table name %q", table.Name)
		}
		if !identifier.MatchString(table.Name) || seenTables[table.Name] || len(table.Columns) == 0 || len(table.Columns) > 30 {
			return fmt.Errorf("invalid or duplicate table %q", table.Name)
		}
		seenTables[table.Name] = true
		columns := map[string]Column{}
		for _, column := range table.Columns {
			if systemColumns[column.Name] {
				return fmt.Errorf("reserved PostgreSQL system column %q in %q", column.Name, table.Name)
			}
			if !identifier.MatchString(column.Name) || !columnTypes[column.Type] || columns[column.Name].Name != "" {
				return fmt.Errorf("invalid column in %q", table.Name)
			}
			columns[column.Name] = column
		}
		if len(table.PrimaryKey) == 0 || len(table.PrimaryKey) > 3 {
			return fmt.Errorf("table %q needs a primary key", table.Name)
		}
		keys := map[string]bool{}
		for _, key := range table.PrimaryKey {
			column, exists := columns[key]
			if !exists || column.Nullable || keys[key] {
				return fmt.Errorf("invalid primary key in %q", table.Name)
			}
			keys[key] = true
		}
		if len(table.Indexes) > 10 {
			return fmt.Errorf("too many indexes in %q", table.Name)
		}
		for _, index := range table.Indexes {
			if len(index.Columns) == 0 || len(index.Columns) > 3 {
				return fmt.Errorf("invalid index in %q", table.Name)
			}
			seen := map[string]bool{}
			for _, name := range index.Columns {
				if _, exists := columns[name]; !exists || seen[name] {
					return fmt.Errorf("invalid index column in %q", table.Name)
				}
				seen[name] = true
			}
		}
		p := table.Policies
		modes := []string{p.Select, p.Insert, p.Update, p.Delete}
		ownRows := false
		for _, mode := range modes {
			if !validMode(mode) {
				return fmt.Errorf("invalid policy mode in %q", table.Name)
			}
			ownRows = ownRows || mode == "own-rows"
		}
		if ownRows {
			column, exists := columns[p.OwnerColumn]
			if !exists || column.Type != "text" || column.Nullable {
				return fmt.Errorf("own-rows requires a non-null text ownerColumn in %q", table.Name)
			}
		} else if p.OwnerColumn != "" {
			return fmt.Errorf("ownerColumn is only valid with own-rows in %q", table.Name)
		}
	}
	// PostgreSQL tables and indexes share one namespace inside the app schema.
	// Catch generated index names that would collide with a user table or index
	// before applying any DDL.
	relations := map[string]string{}
	for _, table := range m.Tables {
		relations[table.Name] = "table " + table.Name
	}
	for _, table := range m.Tables {
		for position := range table.Indexes {
			name := indexName(table.Name, position)
			if other := relations[name]; other != "" {
				return fmt.Errorf("generated index %q for table %q conflicts with %s", name, table.Name, other)
			}
			relations[name] = "index for table " + table.Name
		}
	}
	return nil
}

func (m Manifest) Digest() (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	content, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
