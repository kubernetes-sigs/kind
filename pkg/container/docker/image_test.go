/*
Copyright 2019 The Kubernetes Authors.

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

package docker

import "testing"

func TestSplitImage(t *testing.T) {
	/*
		alpine -> (alpine, latest)

		alpine:latest -> (alpine, latest)

		alpine@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913 -> (alpine, latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913)

		alpine:latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913 -> (alpine, latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913)
	*/
	cases := []struct {
		Image            string
		ExpectedRegistry string
		ExpectedTag      string
		ExpectError      bool
	}{
		{
			Image:            "alpine",
			ExpectedRegistry: "alpine",
			ExpectedTag:      "latest",
			ExpectError:      false,
		},
		{
			Image:            "alpine:latest",
			ExpectedRegistry: "alpine",
			ExpectedTag:      "latest",
			ExpectError:      false,
		},
		{
			Image:            "alpine@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectedRegistry: "alpine",
			ExpectedTag:      "latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectError:      false,
		},
		{
			Image:            "alpine:latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectedRegistry: "alpine",
			ExpectedTag:      "latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectError:      false,
		},
		{
			Image:            "k8s.gcr.io/coredns:1.1.3",
			ExpectedRegistry: "k8s.gcr.io/coredns",
			ExpectedTag:      "1.1.3",
			ExpectError:      false,
		},
		{
			Image:            "k8s.gcr.io/coredns:1.1.3@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectedRegistry: "k8s.gcr.io/coredns",
			ExpectedTag:      "1.1.3@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectError:      false,
		},
		{
			Image:            "k8s.gcr.io/coredns:latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectedRegistry: "k8s.gcr.io/coredns",
			ExpectedTag:      "latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectError:      false,
		},
		{
			Image:            "k8s.gcr.io/coredns@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectedRegistry: "k8s.gcr.io/coredns",
			ExpectedTag:      "latest@sha256:28ef97b8686a0b5399129e9b763d5b7e5ff03576aa5580d6f4182a49c5fe1913",
			ExpectError:      false,
		},
		{
			Image:            ":",
			ExpectedRegistry: "",
			ExpectedTag:      "",
			ExpectError:      true,
		},
		{
			Image:            "@",
			ExpectedRegistry: "",
			ExpectedTag:      "",
			ExpectError:      true,
		},
		{
			Image:            "a@",
			ExpectedRegistry: "",
			ExpectedTag:      "",
			ExpectError:      true,
		},
		{
			Image:            "a:",
			ExpectedRegistry: "",
			ExpectedTag:      "",
			ExpectError:      true,
		},
	}

	for _, tc := range cases {
		tc := tc // capture tc
		t.Run(tc.Image, func(t *testing.T) {
			t.Parallel()

			registry, tag, err := SplitImage(tc.Image)
			if err != nil && !tc.ExpectError {
				t.Fatalf("Unexpected error: %q", err)
			} else if err == nil && tc.ExpectError {
				t.Fatalf("Expected error but got nil")
			}
			if registry != tc.ExpectedRegistry {
				t.Fatalf("ExpectedRegistry %q != %q", tc.ExpectedRegistry, registry)
			}
			if tag != tc.ExpectedTag {
				t.Fatalf("ExpectedTag %q != %q", tc.ExpectedTag, tag)
			}
		})
	}
}
