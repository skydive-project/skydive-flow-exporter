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

/*
 * This code is to support the "action" feature of aws in the vpc flow logs.
 * For each flow, specify whether it was Accepted or Rejected.
 * The assumption is that we have 2 points where the flows are captured: one on each side of a firewall.
 * If a flow appears on only one side of the firewall, we declare it as Rejected.
 * A flow is considered Rejected when we haven't seen that flow on the second interface after a prescribed timeout.
 * If the flow appears on both sides of the firewall, we declare it as Accepted.
 * When a flow is blocked mid-stream, we need to accurately report the number of bytes and packets that were Accepted.
 *
 * The algorithm to implement the feature is as follows.
 * For each accepted flow, we should see the flow logs at both points of capture.
 * The points of capture are characterized by their TIDs.
 * For each rejected flow, we should see the flow logs at only one of the points of capture.
 * For each round of reported flows, match up the pairs of accepted flows using their common Tracking ID.
 * Measurements on the 2 interfaces may happen independently of one another.
 * Since a flow may change from Accepted to Rejected mid-stream, we should be careful to report as Accepted
 * only those bytes and packets that have already been confirmed at both capture points.
 * We therefore report the lesser of the numbers (bytes, packets, time) recorded in the 2 measurements of the same flow.
 */

package pkg

import (
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/skydive-project/skydive-flow-exporter/core"
	"github.com/skydive-project/skydive/common"
	"github.com/skydive-project/skydive/flow"
	"github.com/skydive-project/skydive/logging"
)

// time for cleanup of old connections
const timeoutPeriodCleanup = 20
const defaultRejectTimeout = 5

type combinedFlowInfo struct {
	flows          map[string]*VpclogsFlow // list of flows received for this TrackingID, indexed by UUID
	flowsPrev      map[string]*VpclogsFlow
	flowReported   *VpclogsFlow // combined flow reported to user
	metricReported *flow.FlowMetric
	lastReportTime int64
}

type Action struct {
	// table of saved flows
	flowInfOState     map[string]*combinedFlowInfo
	timeOfLastCleanup int64
	tid1              string
	tid2              string
	rejectTimeout     int64
}

// NewAction - creates data structures needed for vpc Accept/Reject based on measurements of flows at 2 points
func NewAction(cfg *viper.Viper) (interface{}, error) {
	flowInfOState := make(map[string]*combinedFlowInfo)
	tid1 := cfg.GetString(core.CfgRoot + "action.tid1")
	if tid1 == "" {
		logging.GetLogger().Errorf("tid1 not defined")
		return nil, fmt.Errorf("Failed to detect tid1")
	}
	tid2 := cfg.GetString(core.CfgRoot + "action.tid2")
	if tid2 == "" {
		logging.GetLogger().Errorf("tid2 not defined")
		return nil, fmt.Errorf("Failed to detect tid2")
	}
	rejectTimeout := cfg.GetInt64(core.CfgRoot + "action.reject_timeout")
	if rejectTimeout == 0 {
		rejectTimeout = defaultRejectTimeout
	}
	logging.GetLogger().Infof("tid1 = %s", tid1)
	logging.GetLogger().Infof("tid2 = %s", tid2)
	logging.GetLogger().Infof("reject_timeout = %d", rejectTimeout)
	action := &Action{
		flowInfOState:     flowInfOState,
		timeOfLastCleanup: time.Now().Unix(),
		tid1:              tid1,
		tid2:              tid2,
		rejectTimeout:     rejectTimeout,
	}
	return action, nil
}

// resetFlowInfoState - initialize internal data structure for this round
func (t *Action) resetFlowInfoState() {
	// reset the entries in the table for this round
	for _, fis := range t.flowInfOState {
		for _, f := range fis.flows {
			// save copy of latest version of each flow (overwrite earlier versions)
			fis.flowsPrev[f.Flow.UUID] = fis.flows[f.Flow.UUID]
			delete(fis.flows, f.Flow.UUID)
		}
	}
}

// collectFlowInfo - go over all the flows and fill in our internal data structure with relevant info
func (t *Action) collectFlowInfo(fl []interface{}) {
	for _, i := range fl {
		f := i.(*VpclogsFlow)
		// only consider flows that match the specified TIDs
		if f.Flow.NodeTID != t.tid1 && f.Flow.NodeTID != t.tid2 {
			logging.GetLogger().Errorf("TIDs don't match: tid1 = %s, tid2 = %s, flow TID = %s, flowInfo = %s", t.tid1, t.tid2, f.Flow.NodeTID, f)
			continue
		}
		TrackingID := f.Flow.L3TrackingID
		cfi, ok := t.flowInfOState[TrackingID]
		if !ok {
			// this flow is not yet in our table
			cfi = &combinedFlowInfo{
				flows:     make(map[string]*VpclogsFlow),
				flowsPrev: make(map[string]*VpclogsFlow),
			}
			t.flowInfOState[TrackingID] = cfi
		}
		uidEntry, ok := cfi.flows[f.Flow.UUID]
		if !ok {
			// entry does not yet exist
			cfi.flows[f.Flow.UUID] = f
		} else if f.Flow.LastUpdateMetric.Last > uidEntry.Flow.LastUpdateMetric.Last {
			// update entry if time stamp is greater
			cfi.flows[f.Flow.UUID] = f
		}
	}
}

// prepareCombinedMetric initializes a combined FlowMetric structure for the common stats provided
func prepareCombinedMetric(M1, M2 *flow.FlowMetric) *flow.FlowMetric {
	// take byte and packet count as minimum of the 2 reports
	return &flow.FlowMetric{
		ABPackets: common.MinInt64(M1.ABPackets, M2.ABPackets),
		BAPackets: common.MinInt64(M1.BAPackets, M2.BAPackets),
		ABBytes:   common.MinInt64(M1.ABBytes, M2.ABBytes),
		BABytes:   common.MinInt64(M1.BABytes, M2.BABytes),
		Last:      common.MinInt64(M1.Last, M2.Last),
		Start:     M1.Start,
	}
}

// obtainKeys - takes a map with 2 elements and returns the keys of those elements
func obtainKeys(mymap map[string]*VpclogsFlow) (string, string) {
	if len(mymap) != 2 {
		return "", ""
	}
	var keys [2]string
	i := 0
	for k := range mymap {
		keys[i] = k
		i++
	}
	return keys[0], keys[1]
}

// exceedTimeoutThreshold determines whether one of the 2 flows of a combined flow is missing for too long and is determined to have been dropped
func (t *Action) exceedTimeoutThreshold(f *combinedFlowInfo) bool {
	k1, k2 := obtainKeys(f.flows)
	flow1 := f.flows[k1]
	flow2 := f.flows[k2]

	timeDifference := (flow1.Flow.LastUpdateMetric.Last - flow2.Flow.LastUpdateMetric.Last) / 1000
	if timeDifference < 0 {
		timeDifference = -timeDifference
	}
	if timeDifference > t.rejectTimeout {
		return true
	}
	return false
}

// prepareTimeoutResponse - connection timed out; report Reject
// set fields including LastUpdateMetric; byte and packet fields are 0 by initialization
func prepareTimeoutResponse(f *combinedFlowInfo, flow1 *VpclogsFlow) *VpclogsFlow {
	newF := *flow1
	var newLastUpdateMetric *flow.FlowMetric = new(flow.FlowMetric)
	newLastUpdateMetric.Start = f.metricReported.Start
	newLastUpdateMetric.Last = f.metricReported.Last
	newF.LastUpdateMetric = newLastUpdateMetric
	newF.Metric = f.metricReported
	newF.SetAction(VpcActionReject)
	newF.DeriveExternalizedFields()
	f.flowReported = &newF
	f.lastReportTime = time.Now().Unix()
	return &newF
}

// combineFlows - combine 2 seen flows into one, based on common minimums
func (t *Action) combineFlows(f *combinedFlowInfo) *VpclogsFlow {
	// get pointers to the 2 flow structures that make up this combined flow
	k1, k2 := obtainKeys(f.flows)
	flow1 := f.flows[k1]
	flow2 := f.flows[k2]
	logging.GetLogger().Debugf("combineFlows: flow1 = %s", flow1.Flow)
	logging.GetLogger().Debugf("combineFlows: flow2 = %s", flow2.Flow)

	// perform shallow copy of flow and then update some fields
	newF := *flow1
	// verify we received flows from each of the 2 specified interfaces
	tidOK := true
	if flow1.Flow.NodeTID != t.tid1 {
		if flow1.Flow.NodeTID != t.tid2 || flow2.Flow.NodeTID != t.tid1 {
			tidOK = false
		}
	} else if flow2.Flow.NodeTID != t.tid2 {
		tidOK = false
	}
	if !tidOK {
		logging.GetLogger().Errorf("TIDs don't match: tid1 = %s, tid2 = %s, flow1.TID = %s, flow2.TID = %s, flowInfo = %s", t.tid1, t.tid2, flow1.Flow.NodeTID, flow2.Flow.NodeTID, f)
		newF.SetAction(VpcActionReject)
		return &newF
	}
	newMetricReported := prepareCombinedMetric(flow1.Flow.Metric, flow2.Flow.Metric)

	// ensure that we have new data to report
	if f.metricReported != nil && *newMetricReported == *f.metricReported {
		// we have nothing new to report; check if timestamp difference of flows exceeds threshold.
		if t.exceedTimeoutThreshold(f) {
			return prepareTimeoutResponse(f, flow1)
		}
		return nil
	}
	var newLastUpdateMetric *flow.FlowMetric = new(flow.FlowMetric)

	// if we are at the end of the flow, set Last time to latest of the times measured
	if flow1.WasTerminated && flow2.WasTerminated {
		newMetricReported.Last = common.MaxInt64(flow1.Flow.Metric.Last, flow2.Flow.Metric.Last)
	}

	// compute LastUpdateMetric based on previous metricReported and current newMetricReported
	if f.flowReported != nil {
		newLastUpdateMetric.ABPackets = newMetricReported.ABPackets - f.metricReported.ABPackets
		newLastUpdateMetric.ABBytes = newMetricReported.ABBytes - f.metricReported.ABBytes
		newLastUpdateMetric.BAPackets = newMetricReported.BAPackets - f.metricReported.BAPackets
		newLastUpdateMetric.BABytes = newMetricReported.BABytes - f.metricReported.BABytes
		newLastUpdateMetric.Start = f.metricReported.Last
	} else {
		// this is the first report for the flow
		newLastUpdateMetric = prepareCombinedMetric(flow1.Flow.LastUpdateMetric, flow2.Flow.LastUpdateMetric)
		newLastUpdateMetric.Start = newMetricReported.Start
	}
	newLastUpdateMetric.Last = newMetricReported.Last

	newF.Metric = newMetricReported
	newF.LastUpdateMetric = newLastUpdateMetric
	newF.SetAction(VpcActionAccept)
	newF.DeriveExternalizedFields()

	f.metricReported = newMetricReported
	f.flowReported = &newF
	f.lastReportTime = time.Now().Unix()

	logging.GetLogger().Debugf("combineFlows: %s\n\n", newF)
	return &newF
}

// threeFlows - error handling when we receive more than 2 flow reports; with proper setup this should not occur
func (t *Action) threeFlows(f *combinedFlowInfo) *VpclogsFlow {
	var k1 string
	for k := range f.flows {
		k1 = k
		break
	}
	f.flows[k1].SetAction(VpcActionReject)
	f.flowReported = f.flows[k1]
	f.metricReported = f.flows[k1].Metric
	f.lastReportTime = time.Now().Unix()
	logging.GetLogger().Errorf("received 3 flows on tracking ID %s, first UUID = %s", f.flowReported.Flow.TrackingID, f.flowReported.Flow.UUID)
	return f.flows[k1]
}

// oneFlow - only one flow report received on this iteration
// most likely this a Rejected flow and it didn't get through the firewall
// perform additional checks (including timeout) before reporting Reject
func (t *Action) oneFlow(f *combinedFlowInfo) *VpclogsFlow {
	secs := time.Now().Unix()
	if len(f.flowsPrev) == 0 {
		// this is the first appearance of this flow; wait until timeout before reporting reject
		f.lastReportTime = secs
		return nil
	} else if len(f.flowsPrev) == 1 {
		// obtain the keys (UUID) of the current flow and previous flow
		var k1, k2 string
		for k := range f.flows {
			k1 = k
		}
		for k := range f.flowsPrev {
			k2 = k
		}
		// if the UUIDs match, check the timeout
		if k1 == k2 {
			// check for timeout
			if secs > f.lastReportTime+t.rejectTimeout {
				f.flows[k1].SetAction(VpcActionReject)
				f.flowReported = f.flows[k1]
				f.metricReported = f.flows[k1].Metric
				f.lastReportTime = secs
				return f.flows[k1]
			}
			// no timeout; nothing to report on this round
			return nil
		}
		// we have 2 different flows; put them together to make a report
		f.flows[k2] = f.flowsPrev[k2]
		return t.combineFlows(f)
	}
	// we previously had 2 flows; check to see if we need to send an updated metric
	// k1 is the key of the flow currently detected
	var k1 string
	for k := range f.flows {
		k1 = k
	}
	k1Prev, k2Prev := obtainKeys(f.flowsPrev)
	if k1 == k1Prev {
		f.flows[k2Prev] = f.flowsPrev[k2Prev]
	} else {
		f.flows[k1Prev] = f.flowsPrev[k1Prev]
	}
	return t.combineFlows(f)
}

// zeroFlows - no reports were received on this iteration for this flow
// this may be OK for a long-lasting TCP connection that has no traffic for a while
func (t *Action) zeroFlows(f *combinedFlowInfo) *VpclogsFlow {
	// check for timeouts and end of flow
	if len(f.flowsPrev) == 1 {
		secs := time.Now().Unix()
		for _, fl := range f.flowsPrev {
			if secs > f.lastReportTime+t.rejectTimeout {
				fl.SetAction(VpcActionReject)
				f.flowReported = fl
				f.metricReported = fl.Metric
				f.lastReportTime = secs
				return fl
			}
			return nil
		}
	}
	// had 2 prev flows; see if we hit a timeout
	k1, k2 := obtainKeys(f.flowsPrev)
	f.flows[k1] = f.flowsPrev[k1]
	f.flows[k2] = f.flowsPrev[k2]
	if t.exceedTimeoutThreshold(f) {
		return prepareTimeoutResponse(f, f.flowsPrev[k1])
	}
	return nil
}

// cleanupOldEntries - delete old entries from table; allocate a new table and preserve only the live connections
func (t *Action) cleanupOldEntries() {
	// allocate a new status buffer
	flowInfOStateNew := make(map[string]*combinedFlowInfo)
	for index, f := range t.flowInfOState {
		if f.flowReported != nil {
			if f.flowReported.Action == VpcActionReject || f.flowReported.WasTerminated {
				// remove these entries from table
				continue
			}
		}
		flowInfOStateNew[index] = f
	}
	t.flowInfOState = flowInfOStateNew
	t.timeOfLastCleanup = time.Now().Unix()
}

// Action - combines flows from 2 measurement points to determine whether a flow was Accepted or Rejected
func (t *Action) Action(fl []interface{}) []interface{} {
	t.resetFlowInfoState()
	t.collectFlowInfo(fl)

	// go over the table and prepare the items that need to be reported
	// flows that were seen twice are Accepted; flows seen once need special handling for accounting purposes
	var newFl []interface{}
	var f2 *VpclogsFlow
	for _, f := range t.flowInfOState {
		if len(f.flows) == 2 {
			// good case: we received exactly 2 reports, as expected
			f2 = t.combineFlows(f)
		} else if len(f.flows) == 0 {
			f2 = t.zeroFlows(f)
		} else if len(f.flows) == 1 {
			f2 = t.oneFlow(f)
		} else {
			// 3 or more flows
			f2 = t.threeFlows(f)
		}
		// f2 might be null; if flow is valid but did not appear in most recent round, we report nothing
		if f2 != nil {
			f.flowReported = f2
			newFl = append(newFl, f2)
		}
	}
	// delete old Rejected entries every so often
	secs := time.Now().Unix()
	secs2 := t.timeOfLastCleanup + timeoutPeriodCleanup
	if secs > secs2 {
		t.cleanupOldEntries()
	}
	return newFl
}

func (t *Action) Mangle(in []interface{}) []interface{} {
	out := t.Action(in)
	return out
}
