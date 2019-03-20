// +build e2e

/*
Copyright 2018 The Knative Authors

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

package e2e

import (
	"fmt"
	"testing"
	"time"

	// Mysteriously required to support GCP auth (required by k8s libs).
	// Apparently just importing it is enough. @_@ side effects @_@.
	// https://github.com/kubernetes/client-go/issues/242.
	// DO NOT REMOVE.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestIstioScaleToN(t *testing.T) {
	// Run each of these variations.
	tests := []struct {
		size    int
		timeout time.Duration
	}{{
		size:    1,
		timeout: 60 * time.Second,
	}, {
		size:    200,
		timeout: 5 * time.Minute,
	}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.size), func(t *testing.T) {
			IstioScaleToWithin(t, test.size, test.timeout)
		})
	}
}
