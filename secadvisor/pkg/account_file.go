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

package pkg

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/spf13/viper"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/logging"
)

func now() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

type accountFile struct {
	filename string
}

type accountRecord struct {
	Version         string `json:"version"`
	AccumulateStart int64  `json:"accumulate-start"`
	AccumulateEnd   int64  `json:"accumulate-end"`
	AccumulateBytes int64  `json:"accumulate-bytes"`
	CurrentStart    int64  `json:"current-start"`
	CurrentEnd      int64  `json:"current-end"`
	CurrentBytes    int64  `json:"current-bytes"`
}

func newAccountRecord(now int64) *accountRecord {
	return &accountRecord{
		Version:         version,
		AccumulateStart: now,
		AccumulateEnd:   now,
		CurrentStart:    now,
		CurrentEnd:      now,
	}
}

func verifyAccountRecord(record *accountRecord) error {
	if record.Version != version {
		return os.ErrInvalid
	}

	if record.AccumulateStart == 0 || record.AccumulateStart > record.AccumulateEnd {
		return os.ErrInvalid
	}

	if record.CurrentStart == 0 || record.CurrentStart > record.CurrentEnd {
		return os.ErrInvalid
	}

	return nil
}

// read will attempt to read content of file
func (a *accountFile) read(filename string) (*accountRecord, error) {
	now := now()

	file, err := os.Open(filename)
	if err != nil {
		return newAccountRecord(now), err
	}
	defer file.Close()

	var record accountRecord
	content, err := ioutil.ReadAll(file)
	if err != nil {
		logging.GetLogger().Error("read `%s`: %s", filename, err)
		return newAccountRecord(now), err
	}

	err = json.Unmarshal(content, &record)
	if err != nil {
		logging.GetLogger().Error("unmarshal `%s`: %s", filename, err)
		return newAccountRecord(now), err
	}

	if err := verifyAccountRecord(&record); err != nil {
		logging.GetLogger().Error("verify `%s`: %s", filename, err)
		return newAccountRecord(now), os.ErrInvalid
	}

	return &record, nil
}

func (a *accountFile) write(filename string, record *accountRecord) {
	content, err := json.MarshalIndent(record, "", " ")
	if err != nil {
		logging.GetLogger().Error("marshal `%s`: %s", filename, err)
		return
	}
	err = ioutil.WriteFile(filename, content, 0644)
	if err != nil {
		logging.GetLogger().Error("write `%s`: %s", filename, err)
	}
}

func (a *accountFile) bakfile() string {
	return a.filename + ".bak"
}

// Reset counters
func (a *accountFile) Reset() {
	os.Remove(a.filename)
	os.Remove(a.bakfile())
}

// Add to counters
func (a *accountFile) Add(bytes int64) {
	now := now()

	record, err := a.read(a.filename)
	if err != nil {
		record, _ = a.read(a.bakfile())
	}

	record.CurrentStart = now
	record.CurrentEnd = now
	record.CurrentBytes = bytes

	record.AccumulateEnd = now
	record.AccumulateBytes += bytes

	a.write(a.bakfile(), record)
	os.Remove(a.filename)
	if err = os.Rename(a.bakfile(), a.filename); err != nil {
		logging.GetLogger().Error("rename `%s` to `%s`: %s", a.bakfile(), a.filename, err)
	}
}

// NewAccountNone create a new accounter
func NewAccountFile(cfg *viper.Viper) (interface{}, error) {
	filename := cfg.GetString(core.CfgRoot + "account.file.filename")
	return &accountFile{filename: filename}, nil
}
