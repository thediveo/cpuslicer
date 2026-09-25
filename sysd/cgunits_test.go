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
	"os"
	"slices"

	sddbus "github.com/coreos/go-systemd/v22/dbus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("cgroup-related units", Ordered, func() {

	BeforeAll(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
	})

	It("lists all cgroup-related units", func(ctx context.Context) {
		// note: we can use the spec context here for the connection as we don't
		// use it in a deferred cleanup outside the spec.
		sdconn := Successful(sddbus.NewSystemdConnectionContext(ctx))
		DeferCleanup(sdconn.Close)

		Expect(slices.Collect(AllCgroupUnits(ctx, sdconn))).To(ContainElements(
			"-.slice",
			"system.slice",
			"init.scope",
			"dbus.service",
		))
	})

	It("determines the hierarchy of slice units", func(ctx context.Context) {
		// note: we can use the spec context here for the connection as we don't
		// use it in a deferred cleanup outside the spec.
		sdconn := Successful(sddbus.NewSystemdConnectionContext(ctx))
		DeferCleanup(sdconn.Close)

		h := slicesHierarchy(ctx, sdconn)
		Expect(h).NotTo(HaveKey(""))
		Expect(h).To(HaveKeyWithValue("-.slice", ContainElements("system.slice", "user.slice")))
	})

})
