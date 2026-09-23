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

package cpuslicer

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/thediveo/cpus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("CPU slicing", func() {

	It("excludes the last logical CPU from systemd's own usage", func(ctx context.Context) {
		const (
			InitScopeUnit       = "init.scope"
			AllowedCPUsProperty = "AllowedCPUs"
			AllowedCPUsType     = "Scope"
		)

		// Globbits! dbus.NewSystemdConnection is deprecated and
		// dbus.NewSystemdConnectionContext does not use the passed context for
		// controlling dialing but for controlling the lifetime of the
		// connection. This is so ... unexpected ... and not documented at all.
		//
		// And the ProbLLMs can't correctly analyse the static flow but instead
		// immediately stop dead with the result "context aint used in dialing
		// so no problem here".
		sysdconn := Successful(dbus.NewSystemdConnectionContext(context.Background()))
		DeferCleanup(sysdconn.Close)

		allowedCPUsProp := Successful(
			sysdconn.GetUnitTypePropertyContext(ctx, InitScopeUnit, AllowedCPUsType, AllowedCPUsProperty))
		allowedCPUs := AssignableTo[[]uint8](allowedCPUsProp.Value.Value())
		Expect(cpus.SystemDbusSet(allowedCPUs)).NotTo(BeEmpty())
		DeferCleanup(func(ctx context.Context) {
			Expect(sysdconn.SetUnitPropertiesContext(ctx,
				InitScopeUnit, true, *allowedCPUsProp)).To(Succeed())
		})
	})

})
