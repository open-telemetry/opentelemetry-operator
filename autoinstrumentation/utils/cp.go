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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func copyDir(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return os.Mkdir(filepath.Join(dst, relativePath), info.Mode().Perm())
		} else {
			return copyFile(path, filepath.Join(dst, relativePath))
		}
	})

	return err
}

func runCopy() error {
	args := os.Args[1:]

	if len(args) < 2 {
		return errors.New("Not enough arguments given.")
	}

	src := args[len(args)-2]
	dest := args[len(args)-1]

	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	print(args)
	if stat.Mode().IsDir() {
		if len(args) == 3 && args[0] == "-a" {
			err = copyDir(src, dest)
		} else {
			err = fmt.Errorf("Invalid arguments given.")
		}
	} else {
		if len(args) > 2 {
			return errors.New("Too many arguments given.")
		}
		err = copyFile(src, dest)
	}

	if err != nil {
		return err
	}

	fmt.Println("File copied successfully!")
	return nil
}

func main() {
	err := runCopy()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
