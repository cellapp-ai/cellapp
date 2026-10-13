package appdata

import (
	"fmt"
	"reflect"
)

type Plan struct {
	Version             int      `json:"version"`
	Digest              string   `json:"digest"`
	Operations          []string `json:"operations"`
	RequiresDestructive bool     `json:"requiresDestructive"`
	AlreadyApplied      bool     `json:"alreadyApplied"`
}

func PlanManifest(previous *Manifest, next Manifest) (Plan, error) {
	digest, err := next.Digest()
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Version: next.Version, Digest: digest, Operations: []string{}}
	if previous == nil {
		if next.Version != 1 {
			return Plan{}, fmt.Errorf("first manifest version must be 1")
		}
		for _, table := range next.Tables {
			plan.Operations = append(plan.Operations, "create table "+table.Name)
		}
		return plan, nil
	}
	if err := previous.Validate(); err != nil {
		return Plan{}, fmt.Errorf("stored manifest invalid: %w", err)
	}
	if next.Version == previous.Version {
		oldDigest, err := previous.Digest()
		if err != nil || digest != oldDigest {
			return Plan{}, fmt.Errorf("version already has different content")
		}
		plan.AlreadyApplied = true
		return plan, nil
	}
	if next.Version != previous.Version+1 {
		return Plan{}, fmt.Errorf("manifest version must increase by one")
	}
	oldTables := map[string]Table{}
	newTables := map[string]Table{}
	for _, table := range previous.Tables {
		oldTables[table.Name] = table
	}
	for _, table := range next.Tables {
		newTables[table.Name] = table
		old, exists := oldTables[table.Name]
		if !exists {
			plan.Operations = append(plan.Operations, "create table "+table.Name)
			continue
		}
		if !reflect.DeepEqual(old.PrimaryKey, table.PrimaryKey) {
			return Plan{}, fmt.Errorf("changing primary keys is unsupported in %q", table.Name)
		}
		oldColumns := map[string]Column{}
		newColumns := map[string]Column{}
		for _, column := range old.Columns {
			oldColumns[column.Name] = column
		}
		for _, column := range table.Columns {
			newColumns[column.Name] = column
			prior, exists := oldColumns[column.Name]
			if !exists {
				plan.Operations = append(plan.Operations, "add column "+table.Name+"."+column.Name)
			} else if prior != column {
				return Plan{}, fmt.Errorf("changing column type or nullability is unsupported in %q", table.Name)
			}
		}
		for _, column := range old.Columns {
			if _, exists := newColumns[column.Name]; !exists {
				plan.Operations = append(plan.Operations, "drop column "+table.Name+"."+column.Name)
				plan.RequiresDestructive = true
			}
		}
		if !reflect.DeepEqual(old.Indexes, table.Indexes) {
			plan.Operations = append(plan.Operations, "replace indexes "+table.Name)
		}
		if old.Policies != table.Policies {
			plan.Operations = append(plan.Operations, "update policies "+table.Name)
		}
	}
	for _, table := range previous.Tables {
		if _, exists := newTables[table.Name]; !exists {
			plan.Operations = append(plan.Operations, "drop table "+table.Name)
			plan.RequiresDestructive = true
		}
	}
	if plan.RequiresDestructive && !next.AllowDestructive {
		return Plan{}, fmt.Errorf("destructive changes require allowDestructive=true")
	}
	return plan, nil
}
