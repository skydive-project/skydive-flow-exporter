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

package main

import (
	"github.com/skydive-project/skydive-flow-exporter/core"
	secadvisor "github.com/skydive-project/skydive-flow-exporter/secadvisor/pkg"
)

func main() {
	core.Main("/etc/skydive/secadvisor.yml")
}

func init() {
	core.ClassifierHandlers.Register("subnet_autodiscovery", secadvisor.NewClassifySubnetWithAutoDiscovery, false)
	core.ManglerHandlers.Register("logstatus", secadvisor.NewMangleLogStatus, false)
	core.EncoderHandlers.Register("secadvisor", secadvisor.NewEncode, true)
	core.TransformerHandlers.Register("secadvisor", secadvisor.NewTransform, false)
}
