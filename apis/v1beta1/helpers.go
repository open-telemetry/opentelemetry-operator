// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// parseAddressEndpoint parses the address and returns the host and port.
// If the address is an environment variable, it returns the default port.
// If the address is an explicit port, it returns the port.
// If the address is not valid, it returns an error.
func parseAddressEndpoint(address string) (string, int32, error) {
	// The regex below matches on strings that end with a colon followed by the environment variable expansion syntax.
	// So it should match on strings ending with: ":${env:POD_IP}" or ":${POD_IP}".
	const portEnvVarRegex = `:\${[env:]?.*}$`
	isPortEnvVar := regexp.MustCompile(portEnvVarRegex).MatchString(address)
	if isPortEnvVar {
		errMsg := fmt.Sprintf("couldn't determine metrics port from configuration: %s", address)
		return "", 0, errors.New(errMsg)
	}

	// The regex below matches on strings that end with a colon followed by 1 or more numbers (representing the port).
	const explicitPortRegex = `:(\d+$)`
	explicitPortMatches := regexp.MustCompile(explicitPortRegex).FindStringSubmatch(address)
	if len(explicitPortMatches) <= 1 {
		return address, defaultServicePort, nil
	}

	port, err := strconv.ParseInt(explicitPortMatches[1], 10, 32)
	if err != nil {
		errMsg := fmt.Sprintf("couldn't determine metrics port from configuration: %s", address)
		return "", 0, errors.New(errMsg)
	}

	host, _, _ := strings.Cut(address, explicitPortMatches[0])
	return host, intToInt32Safe(int(port)), nil
}

// addPrefix adds a prefix to each element of the array.
func addPrefix(prefix string, arr []string) []string {
	if len(arr) == 0 {
		return []string{}
	}
	var prefixed []string
	for _, v := range arr {
		prefixed = append(prefixed, fmt.Sprintf("%s%s", prefix, v))
	}
	return prefixed
}

// intToInt32Safe converts an int to an int32.
func intToInt32Safe(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// getNullValuedKeys returns keys from the input map whose values are nil. Keys from nested maps are prefixed
// by the name of the parent key, using a dot notation.
func getNullValuedKeys(cfg map[string]interface{}) []string {
	var nullKeys []string
	for k, v := range cfg {
		if v == nil {
			nullKeys = append(nullKeys, fmt.Sprintf("%s:", k))
		}
		if reflect.ValueOf(v).Kind() == reflect.Map {
			var nulls []string
			val, ok := v.(map[string]interface{})
			if ok {
				nulls = getNullValuedKeys(val)
			}
			if len(nulls) > 0 {
				prefixed := addPrefix(k+".", nulls)
				nullKeys = append(nullKeys, prefixed...)
			}
		}
	}
	return nullKeys
}

// normalizeConfig fixes the config to be valid for the collector.
// It removes nil values, converts float64 to int32.
func normalizeConfig(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case nil:
			// We remove those fields which value is nil. This prevents issues when
			// unmarshalling the config in the collector
			delete(m, k)
		case map[string]interface{}:
			normalizeConfig(val)
		case []interface{}:
			for i, item := range val {
				if item == nil {
					val[i] = map[string]interface{}{}
				} else if sub, ok := item.(map[string]interface{}); ok {
					normalizeConfig(sub)
				}
			}
		case float64:
			// All numbers (even int32, int64, etc.) are parsed as float64 by default
			// We need to convert them to the correct type
			if k == "port" {
				m[k] = int32(val)
			}
		default:
		}
	}
}
