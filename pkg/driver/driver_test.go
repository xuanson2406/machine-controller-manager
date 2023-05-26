/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/xuanson2406/machine-controller-manager/pkg/driver"
)

var _ = Describe("Driver", func() {
	Describe("#ExtractCredentialsFromData", func() {
		It("should return an empty string because data map is nil", func() {
			Expect(ExtractCredentialsFromData(nil, "foo", "bar")).To(BeEmpty())
		})

		It("should return an empty string because data map is empty", func() {
			Expect(ExtractCredentialsFromData(map[string][]byte{}, "foo", "bar")).To(BeEmpty())
		})

		It("should return an empty string because no keys provided", func() {
			Expect(ExtractCredentialsFromData(map[string][]byte{"foo": []byte("bar")})).To(BeEmpty())
		})

		It("should return the value of the first matching key", func() {
			Expect(ExtractCredentialsFromData(map[string][]byte{"foo": []byte(" bar")}, "foo", "bar")).To(Equal("bar"))
		})

		It("should return the value of the first matching key", func() {
			Expect(ExtractCredentialsFromData(map[string][]byte{"foo": []byte(`  bar   
`)}, "bar", "foo")).To(Equal("bar"))
		})
	})
})