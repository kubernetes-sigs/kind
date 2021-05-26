/*
Copyright 2021 The Kubernetes Authors.

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

package podman

import (
	"testing"
)

func Test_sanitizeImage(t *testing.T) {

	cases := []struct {
		image             string
		friendlyImageName string
		pullImageName     string
	}{
		{
			image:             "kindest/node:v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "kindest/node:v1.21.1",
			pullImageName:     "docker.io/kindest/node@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "kindest/node:v1.21.1",
			friendlyImageName: "kindest/node:v1.21.1",
			pullImageName:     "docker.io/kindest/node:v1.21.1",
		},
		{
			image:             "kindest/node@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "kindest/node",
			pullImageName:     "docker.io/kindest/node@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "foo.bar/kindest/node:v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "foo.bar/kindest/node:v1.21.1",
			pullImageName:     "foo.bar/kindest/node@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "foo.bar/kindest/node:v1.21.1",
			friendlyImageName: "foo.bar/kindest/node:v1.21.1",
			pullImageName:     "foo.bar/kindest/node:v1.21.1",
		},
		{
			image:             "foo.bar/kindest/node@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "foo.bar/kindest/node",
			pullImageName:     "foo.bar/kindest/node@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "foo.bar/baz:quux@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "foo.bar/baz:quux",
			pullImageName:     "foo.bar/baz@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "foo.bar/baz:quux",
			friendlyImageName: "foo.bar/baz:quux",
			pullImageName:     "foo.bar/baz:quux",
		},
		{
			image:             "foo.bar/baz@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "foo.bar/baz",
			pullImageName:     "foo.bar/baz@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "baz:quux@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "baz:quux",
			pullImageName:     "docker.io/library/baz@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
		{
			image:             "baz:quux",
			friendlyImageName: "baz:quux",
			pullImageName:     "docker.io/library/baz:quux",
		},
		{
			image:             "baz@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
			friendlyImageName: "baz",
			pullImageName:     "docker.io/library/baz@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6",
		},
	}

	for _, tc := range cases {
		tc := tc // capture variable
		t.Run(tc.image, func(t *testing.T) {
			t.Parallel()
			friendlyImageName, pullImageName := sanitizeImage(tc.image)
			if friendlyImageName != tc.friendlyImageName {
				t.Errorf("Wrong friendlyImageName from %v: expected %v, received %v", tc.image, tc.friendlyImageName, friendlyImageName)
			}
			if pullImageName != tc.pullImageName {
				t.Errorf("Wrong pullImageName from %v: expected %v, received %v", tc.image, tc.pullImageName, pullImageName)
			}
		})
	}
}
