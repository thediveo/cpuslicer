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

package cgunits

import (
	"context"
	"iter"
	"strings"

	sddbus "github.com/coreos/go-systemd/v22/dbus"
)

// IsSlice returns true if the unit name indicates a slice unit.
func IsSlice(unit string) bool { return strings.HasSuffix(unit, ".slice") }

// IsService returns true if the unit name indicates a (leaf) service unit.
func IsService(unit string) bool { return strings.HasSuffix(unit, ".service") }

// IsScope returns true if the unit name indicates a (leaf) scope unit.
func IsScope(unit string) bool { return strings.HasSuffix(unit, ".scope") }

var unitTypes = map[string]string{
	"scope":   "Scope",
	"service": "Service",
	"slice":   "Slice",
}

// Type returns the unit type or D-Bus object interface name for the given unit
// name, based on its suffix.
func Type(unit string) string {
	dot := strings.LastIndex(unit, ".")
	if dot < 0 {
		return ""
	}
	return unitTypes[unit[dot+1:]]
}

// UnitType is a cgroup-related unit type of SliceType, ServiceType, ScopeType.
type UnitType int

const (
	SliceType UnitType = 1 << iota
	ServiceType
	ScopeType
)

// All returns an iterator over all names of cgroup-related systemd units of
// types slice, service, and scope. It silently ignores errors, not yielding any
// unit names.
func All(ctx context.Context, sdconn *sddbus.Conn) iter.Seq[string] {
	return OfType(ctx, sdconn, SliceType|ServiceType|ScopeType)
}

// OfType returns an iterator over all names of cgroup-related systemd units of
// the specified type(s).
func OfType(ctx context.Context, sdconn *sddbus.Conn, types UnitType) iter.Seq[string] {
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

// Hierarchy maps slice unit names to the names of their child slice units, as
// well as services and scope unit names. The keys of this map are solely slice
// unit names, but never service and scope unit names; these can only appear as
// values.
type Hierarchy map[string][]string

// GetHierarchy returns the hierarchy of slice units as well as their scope and
// service units leafs.
func GetHierarchy(ctx context.Context, sdconn *sddbus.Conn) Hierarchy {
	return getHierarchy(ctx, sdconn, true)
}

// GetSlicesHierarchy returns the hierarchy of slice units.
func GetSlicesHierarchy(ctx context.Context, sdconn *sddbus.Conn) Hierarchy {
	return getHierarchy(ctx, sdconn, false)
}

func getHierarchy(ctx context.Context, sdconn *sddbus.Conn, leaves bool) Hierarchy {
	types := SliceType
	if leaves {
		types |= ScopeType | ServiceType
	}
	h := Hierarchy{}
	for unit := range OfType(ctx, sdconn, types) {
		if IsSlice(unit) {
			// All slice units have a Slice property in their Slice interface,
			// telling us the parent slice of this slice unit.
			sliceProp, err := sdconn.GetUnitTypePropertyContext(ctx, unit, "Slice", "Slice")
			if err != nil {
				continue
			}
			slice, _ := sliceProp.Value.Value().(string)
			if slice == "" {
				// If we could not read the property => skip this unit as we have no
				// idea where to place it in the hierarchy. If this is the root
				// slice unit, it has no parent slice and we simply skip it too,
				// knowing that there *are* child slices (not least "init.scope")
				// that will mention the root slice unit and thus will ensure that
				// the root slice unit key gets created correctly.
				continue
			}
			// since the units are returned "unordered" (whatever "ordered"
			// would actually mean we better not contemplate) we might end up
			// seeing a parent scope unit after one of its child scope units, so
			// we need to conditionally initialize in order to not clobber an
			// already existing parent scope unit key with its children unit
			// value.
			if _, ok := h[unit]; !ok {
				h[unit] = []string{}
			}
			h[slice] = append(h[slice], unit)
			continue
		}

		// We're dealing with a leaf cgroup-related unit, that is, a service or
		// scope. These are never added as map keys, only as values of their
		// scope unit.
		sliceProp, err := sdconn.GetUnitTypePropertyContext(ctx, unit, Type(unit), "Slice")
		if err != nil {
			continue
		}
		slice, _ := sliceProp.Value.Value().(string)
		if slice == "" {
			// If we could not read the property => skip this unit as we have no
			// idea where to place it in the hierarchy.
			continue
		}
		h[slice] = append(h[slice], unit)
	}
	return h
}
