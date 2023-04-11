// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name          string
	args          []string
	expectedError bool
}

func TestCopy(t *testing.T) {
	testCases := []testCase{
		{
			name:          "copy single file successfully",
			args:          []string{"src_dir/file1.txt", "dest_dir/file1.txt"},
			expectedError: false,
		},
		{
			name:          "copy single file with incorrect source path",
			args:          []string{"src_dir/missing-file1.txt", "dest_dir/file1_copy.txt"},
			expectedError: true,
		},
		{
			name:          "copy single file with incorrect destination path",
			args:          []string{"src_dir/file1.txt", "missing_dest_dir/file1_copy.txt"},
			expectedError: true,
		},
		{
			name:          "copy directory successfully",
			args:          []string{"-a", "src_dir", "dest_dir/src_dir_copy"},
			expectedError: false,
		},
		{
			name:          "copy directory with incorrect source path",
			args:          []string{"-a", "missing_src_dir", "dest_dir/src_dir_copy"},
			expectedError: true,
		},
		{
			name:          "copy directory with incorrect destination path",
			args:          []string{"-a", "src_dir", "missing-dest_dir/src_dir_copy"},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "example")
			if err != nil {
				t.Fatal(err)
			}
			srcDir := filepath.Join(tempDir, "src_dir")
			err = os.Mkdir(srcDir, 0700)
			if err != nil {
				t.Fatal(err)
			}
			err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0600)
			if err != nil {
				t.Fatal(err)
			}
			if tc.args[0] == "-a" {
				srcDir2 := filepath.Join(tempDir, "src_dir/src_dir2")
				err = os.Mkdir(srcDir2, 0700)
				if err != nil {
					t.Fatal(err)
				}
				err = os.WriteFile(filepath.Join(srcDir2, "file2.txt"), []byte("world"), 0600)
				if err != nil {
					t.Fatal(err)
				}
			}
			destDir := filepath.Join(tempDir, "dest_dir")
			err = os.Mkdir(destDir, 0700)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			tc.args[len(tc.args)-2] = filepath.Join(tempDir, tc.args[len(tc.args)-2])
			tc.args[len(tc.args)-1] = filepath.Join(tempDir, tc.args[len(tc.args)-1])
			tc.args = append([]string{"./cp"}, tc.args...) // ["cp", "-a", "/src", "/dest"]
			os.Args = tc.args
			runCopy()

			if len(tc.args) == 3 && !tc.expectedError {
				source, err := os.ReadFile(tc.args[1])
				assert.NoError(t, err)

				destination, err := os.ReadFile(tc.args[2])
				assert.NoError(t, err)

				assert.Equal(t, source, destination)
			} else if len(tc.args) == 4 && tc.args[1] == "-a" && !tc.expectedError {
				err = filepath.Walk(tc.args[2], func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					relativePath, err := filepath.Rel(tc.args[2], path)
					if err != nil {
						return err
					}
					expectedPath := filepath.Join(tc.args[3], relativePath)

					if !info.IsDir() {
						source, err := os.ReadFile(path)
						assert.NoError(t, err)
						destination, err := os.ReadFile(expectedPath)
						assert.NoError(t, err)
						assert.Equal(t, source, destination)
					}

					return nil
				})
				assert.NoError(t, err)
			}
		})
	}
}
