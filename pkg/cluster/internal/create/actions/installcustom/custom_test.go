package installcustom

import (
	"fmt"
	"os"
	"testing"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/internal/assert"
)

func TestAddCustomManifests(t *testing.T) {
	cases := []struct {
		TestName       string
		CustomManifest []interface{}
		CreateFiles    map[string]string
		ExpectError    string
		ExpectOutput   []map[string]string
		OutputError    string
	}{
		{
			TestName: "Correct inline manifest",
			CustomManifest: []interface{}{
				map[string]string{
					"test2.yaml": "test2: test",
				},
				map[string]interface{}{
					"test3.yaml": "test3: test",
				},
			},
			ExpectOutput: []map[string]string{
				{"-": "test2: test"},
				{"-": "test3: test"},
			},
		},
		{
			TestName: "Correct file manifest",
			CustomManifest: []interface{}{
				"correct_manifest1.yaml",
			},
			CreateFiles: map[string]string{
				"correct_manifest1.yaml": "test: test",
			},
			ExpectOutput: []map[string]string{
				{"-": "test: test"},
			},
		},
		{
			TestName: "Remote http file manifest",
			CustomManifest: []interface{}{
				"https://test.local/test.yaml",
			},
			ExpectOutput: []map[string]string{
				{"https://test.local/test.yaml": ""},
			},
		},
		{
			TestName: "kubectl error",
			CustomManifest: []interface{}{
				map[string]string{
					"test2.yaml": "test2: test",
				},
			},
			ExpectError: "customManifest[0][test2.yaml]: error deploying manifest: error",
			OutputError: "error",
			ExpectOutput: []map[string]string{
				{"-": "test2: test"},
			},
		},
		{
			TestName: "Non existent file manifest",
			CustomManifest: []interface{}{
				"no_file.yaml",
			},
			ExpectError: "customManifests[0]: 'no_file.yaml' does not exist",
		},
		{
			TestName: "Incorrect manifest type map[string]int",
			CustomManifest: []interface{}{
				map[string]int{
					"test2.yaml": 5,
				},
			},
			ExpectError: "customManifests[0]: incorrect type (map[string]int) expected string or map[string]string",
		},
		{
			TestName: "Incorrect manifest type map[string]interface",
			CustomManifest: []interface{}{
				map[string]interface{}{
					"test2.yaml": 5,
				},
			},
			ExpectError: "customManifests[test2.yaml]: incorrect type (map[string]int) expected string or map[string]string",
		},
		{
			TestName: "Incorrect manifest type multiple",
			CustomManifest: []interface{}{
				5,
				map[int]string{
					6: "hello",
				},
			},
			ExpectError: "customManifests[0]: incorrect type (int) expected string or map[string]string",
		},
	}

	for _, tc := range cases {
		tc := tc //capture loop variable
		t.Run(tc.TestName, func(t *testing.T) {
			// create files for test if required
			if tc.CreateFiles != nil && len(tc.CreateFiles) > 0 {
				for fileName, contents := range tc.CreateFiles {
					err := os.WriteFile(fileName, []byte(contents), 0644)
					if err != nil {
						t.Errorf("unexpected error in creating file %s: %v", fileName, err)
					}
				}
			}

			// override apply manifest function to capture output for test expectations
			expectedOuputIndex := 0
			runApplyCustomManifest = func(controlPlane nodes.Node, path string, stdin string) error {
				expectedStdin, ok := tc.ExpectOutput[expectedOuputIndex][path]
				assert.BoolEqual(t, true, ok)
				assert.StringEqual(t, expectedStdin, stdin)
				expectedOuputIndex++

				if tc.OutputError != "" {
					return fmt.Errorf("%s", tc.OutputError)
				}

				return nil
			}

			err := addCustomManifests(nil, &tc.CustomManifest)

			// check all expected output was produced
			if expectedOuputIndex != len(tc.ExpectOutput) {
				t.Errorf("Test failed, did not reach expected number of outputs, got %d and expected %d", expectedOuputIndex, len(tc.ExpectOutput))
			}

			// the error can be:
			// - nil, in which case we should expect no errors or fail
			if err == nil && len(tc.ExpectError) > 0 {
				t.Errorf("Test failed, unexpected error: %s", tc.ExpectError)
			}

			if err != nil && err.Error() != tc.ExpectError {
				t.Errorf("Test failed, error: %s expected error: %s", err, tc.ExpectError)
			}

			// remove any created test files
			if tc.CreateFiles != nil && len(tc.CreateFiles) > 0 {
				for fileName := range tc.CreateFiles {
					os.Remove(fileName)
				}
			}
		})
	}
}
