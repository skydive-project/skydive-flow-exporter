/*
 * Copyright (C) 2019 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy ofthe License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specificlanguage governing permissions and
 * limitations under the License.
 *
 */

package core

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
)

func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Equal assertion failed: (expected: %T:%#v, actual: %T:%#v)", expected, expected, actual, actual)
	}
}

func AssertEqualInt64(t *testing.T, expected, actual int64) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Equal assertion failed: (expected: %v, actual: %v)", expected, actual)
	}
}

func ConfigFromJSON(t *testing.T, content []byte) *viper.Viper {
	cfg := viper.New()
	cfg.SetConfigType("yaml")
	r := bytes.NewBuffer(content)
	if err := cfg.ReadConfig(r); err != nil {
		t.Fatalf("Failed to read config: %s", err)
	}
	return cfg
}
