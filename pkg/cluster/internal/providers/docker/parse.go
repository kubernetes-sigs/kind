/*
Copyright 2020 The Kubernetes Authors.

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

import (
	"time"
)

/*
Docker CLI outputs time.Time objects with the default string format
This is going to be a huge pain if go actually makes good on their threat
that this format is not stable

see: https://golang.org/pkg/time/#Time.String
*/

const goDefaultTimeFormat = "2006-01-02 15:04:05.999999999 -0700 MST"

type goDefaultTime time.Time

func (g *goDefaultTime) UnmarshalJSON(p []byte) error {
	t, err := time.Parse(`"`+goDefaultTimeFormat+`"`, string(p))
	if err != nil {
		return err
	}
	*g = goDefaultTime(t)
	return nil
}
