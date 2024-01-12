// Copyright 2021 ETH Zurich
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package new_pan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathsMRU(t *testing.T) {
	const maxSize = 3
	cases := []struct {
		name   string
		before []PathFingerprint
		insert PathFingerprint
		after  []PathFingerprint
	}{
		{
			name:   "nil",
			before: nil,
			insert: "a",
			after:  []PathFingerprint{"a"},
		},
		{
			name:   "empty",
			before: []PathFingerprint{},
			insert: "a",
			after:  []PathFingerprint{"a"},
		},
		{
			name:   "new, not full",
			before: []PathFingerprint{"a", "b"},
			insert: "c",
			after:  []PathFingerprint{"c", "a", "b"},
		},
		{
			name:   "existing, not full",
			before: []PathFingerprint{"a", "b"},
			insert: "b",
			after:  []PathFingerprint{"b", "a"},
		},
		{
			name:   "new, full",
			before: []PathFingerprint{"a", "b", "c"},
			insert: "d",
			after:  []PathFingerprint{"d", "a", "b"},
		},
		{
			name:   "existing, full, first",
			before: []PathFingerprint{"a", "b", "c"},
			insert: "a",
			after:  []PathFingerprint{"a", "b", "c"},
		},
		{
			name:   "existing, full, middle",
			before: []PathFingerprint{"a", "b", "c"},
			insert: "b",
			after:  []PathFingerprint{"b", "a", "c"},
		},
		{
			name:   "existing, full, last",
			before: []PathFingerprint{"a", "b", "c"},
			insert: "c",
			after:  []PathFingerprint{"c", "a", "b"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			paths := testdataPathsFromFingerprints(c.before)
			l := pathsMRU(paths)
			l.insert(&Path{Fingerprint: c.insert}, maxSize)
			actual := fingerprintsFromTestdataPaths(l)
			assert.Equal(t, c.after, actual)
		})
	}
}
