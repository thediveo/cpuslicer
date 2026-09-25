// Copyright 2026 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package sysd

import (
	"context"
	"iter"

	sddbus "github.com/coreos/go-systemd/v22/dbus"
)

type CgroupUnitType int

const (
	SliceType CgroupUnitType = 1 << iota
	ServiceType
	ScopeType
)

// AllCgroupUnits returns an iterator over all names of cgroup-related systemd
// units of types slice, service, and scope. It silently ignores errors, not
// yielding any unit names.
func AllCgroupUnits(ctx context.Context, sdconn *sddbus.Conn) iter.Seq[string] {
	return CgroupUnitsOfType(ctx, sdconn, SliceType|ServiceType|ScopeType)
}

func CgroupUnitsOfType(ctx context.Context, sdconn *sddbus.Conn, types CgroupUnitType) iter.Seq[string] {
	patterns := make([]string, 0, 3)
	if types&SliceType != 0 {
		patterns = append(patterns, "*.slice")
	}
	if types&ServiceType != 0 {
		patterns = append(patterns, "*.service")
	}
	if types&ScopeType != 0 {
		patterns = append(patterns, "*.scope")
	}
	return func(yield func(string) bool) {
		units, _ := sdconn.ListUnitsByPatternsContext(ctx, nil, patterns)
		for _, unit := range units {
			if !yield(unit.Name) {
				return
			}
		}
	}
}

func WalkAllCgroupUnits(ctx context.Context, sdconn *sddbus.Conn) iter.Seq[string] {
	units, _ := sdconn.ListUnitsByPatternsContext(ctx,
		nil, []string{"*.slice", "*.service", "*.scope"})
	return func(yield func(string) bool) {
		_ = units // FIXME:
	}
}

// SlicesHierarchy maps slice unit names to the slice unit names of their
// children slice unit names.
type SlicesHierarchy map[string][]string

// slicesHierarchy returns the slice units hierarchy.
func slicesHierarchy(ctx context.Context, sdconn *sddbus.Conn) SlicesHierarchy {
	h := SlicesHierarchy{}
	for unit := range CgroupUnitsOfType(ctx, sdconn, SliceType) {
		if _, ok := h[unit]; !ok {
			h[unit] = []string{}
		}
		sliceProp, err := sdconn.GetUnitTypePropertyContext(ctx, unit, "Slice", "Slice")
		if err != nil {
			continue
		}
		parent, _ := sliceProp.Value.Value().(string)
		if parent == "" {
			continue // also skip root
		}
		h[parent] = append(h[parent], unit)
	}
	return h
}
