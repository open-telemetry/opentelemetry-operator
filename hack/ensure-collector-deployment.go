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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

func removeDevelopment(examplePath string) {
	cmd := exec.Command(
		"kubectl", "delete", "-f", examplePath, "--ignore-not-found=true",
	)
	if err := cmd.Start(); err != nil {
		fmt.Println("Something failed while cleaning the evironment. Ignoring")
	}
}

func main() {
	timeout := 300
	var err error
	if len(os.Args) > 1 {
		timeout, err = strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Println(os.Args[1], "could not be parsed to a number")
			os.Exit(1)
		}
	}

	// Wait for the OpenTelemetry Operator
	fmt.Println("Wait until the OTEL Operator deployment is ready")
	timeoutParam := fmt.Sprintf("--timeout=%ds", timeout)
	cmd := exec.Command(
		"kubectl",
		"wait",
		"--for=condition=available",
		"deployment", "opentelemetry-operator-controller-manager",
		"-n", "opentelemetry-operator-system",
		timeoutParam,
	)

	if errRun := cmd.Run(); err != nil {
		fmt.Println("Error waiting to the OTEL Operator deployment: ", errRun)
		os.Exit(1)
	}

	// Sometimes, the deployment of the OTEL Operator is ready but, when
	// creating new instances of the OTEL Collector, the webhook is not reachable
	// and kubectl apply fails. This code executes kubectl apply to deploy an
	// OTEL Collector until success (or timeout)
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	examplePath := filepath.Join(
		cwd, "tests", "e2e", "smoke-simplest", "00-install.yaml",
	)

	defer removeDevelopment(examplePath)

	fmt.Println("Wait until the creation of OTEL Collectors is available")
	pollInterval := 3 * time.Second
	timeoutPoll := time.Duration(timeout) * time.Second
	err = wait.Poll(pollInterval, timeoutPoll, func() (done bool, err error) {
		cmd := exec.Command(
			"kubectl", "apply", "-f", examplePath,
		)

		if errRun := cmd.Run(); errRun != nil {
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
