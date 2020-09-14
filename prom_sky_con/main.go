/*
 * Copyright (C) 2020 IBM, Inc.
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
	prom "github.com/skydive-project/skydive-flow-exporter/prom_sky_con/pkg"
)

func main() {
	core.Main("/etc/skydive/prom_sky_con.yml")
}

func init() {
	core.StorerHandlers.Register("prom_sky_con", prom.NewStorePrometheusWrapper, false)
}
