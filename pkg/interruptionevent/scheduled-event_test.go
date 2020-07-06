// Copyright 2016-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package interruptionevent_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-node-termination-handler/pkg/ec2metadata"
	"github.com/aws/aws-node-termination-handler/pkg/interruptionevent"
	h "github.com/aws/aws-node-termination-handler/pkg/test"
)

const (
	scheduledEventId              = "instance-event-0d59937288b749b32"
	scheduledEventState           = "active"
	scheduledEventCode            = "system-reboot"
	scheduledEventStartTime       = "21 Jan 2019 09:00:43 GMT"
	expScheduledEventStartTimeFmt = "2019-01-21 09:00:43 +0000 UTC"
	scheduledEventEndTime         = "21 Jan 2019 09:17:23 GMT"
	expScheduledEventEndTimeFmt   = "2019-01-21 09:17:23 +0000 UTC"
	scheduledEventDescription     = "scheduled reboot"
	imdsV2TokenPath               = "/latest/api/token"
)

var scheduledEventResponse = []byte(`[{
	"NotBefore": "` + scheduledEventStartTime + `",
	"Code": "` + scheduledEventCode + `",
	"Description": "` + scheduledEventDescription + `",
	"EventId": "` + scheduledEventId + `",
	"NotAfter": "` + scheduledEventEndTime + `",
	"State": "` + scheduledEventState + `"
}]`)

func TestMonitorForScheduledEventsSuccess(t *testing.T) {
	var requestPath string = ec2metadata.ScheduledEventPath

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if imdsV2TokenPath == req.URL.String() {
			rw.WriteHeader(403)
			return
		}
		h.Equals(t, req.URL.String(), requestPath)
		rw.Write(scheduledEventResponse)
	}))
	defer server.Close()

	drainChan := make(chan interruptionevent.InterruptionEvent)
	cancelChan := make(chan interruptionevent.InterruptionEvent)
	imds := ec2metadata.New(server.URL, 1)

	go func() {
		result := <-drainChan
		h.Equals(t, scheduledEventId, result.EventID)
		h.Equals(t, interruptionevent.ScheduledEventKind, result.Kind)
		h.Equals(t, scheduledEventState, result.State)
		h.Equals(t, expScheduledEventStartTimeFmt, result.StartTime.String())
		h.Equals(t, expScheduledEventEndTimeFmt, result.EndTime.String())

		h.Assert(t, strings.Contains(result.Description, scheduledEventCode),
			"Expected description to contain \""+scheduledEventCode+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventStartTime),
			"Expected description to contain \""+scheduledEventStartTime+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventEndTime),
			"Expected description to contain \""+scheduledEventEndTime+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventDescription),
			"Expected description to contain \""+scheduledEventDescription+
				"\"but received \""+result.Description+"\"")

	}()

	err := interruptionevent.MonitorForScheduledEvents(drainChan, cancelChan, imds)
	h.Ok(t, err)
}

func TestMonitorForScheduledEventsCanceledEvent(t *testing.T) {
	var requestPath string = ec2metadata.ScheduledEventPath
	var state = "canceled"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if imdsV2TokenPath == req.URL.String() {
			rw.WriteHeader(403)
			return
		}
		h.Equals(t, req.URL.String(), requestPath)
		rw.Write([]byte(`[{
			"NotBefore": "` + scheduledEventStartTime + `",
			"Code": "` + scheduledEventCode + `",
			"Description": "` + scheduledEventDescription + `",
			"EventId": "` + scheduledEventId + `",
			"NotAfter": "` + scheduledEventEndTime + `",
			"State": "` + state + `"
		}]`))
	}))
	defer server.Close()

	drainChan := make(chan interruptionevent.InterruptionEvent)
	cancelChan := make(chan interruptionevent.InterruptionEvent)
	imds := ec2metadata.New(server.URL, 1)

	go func() {
		result := <-cancelChan
		h.Equals(t, scheduledEventId, result.EventID)
		h.Equals(t, interruptionevent.ScheduledEventKind, result.Kind)
		h.Equals(t, state, result.State)
		h.Equals(t, expScheduledEventStartTimeFmt, result.StartTime.String())
		h.Equals(t, expScheduledEventEndTimeFmt, result.EndTime.String())

		h.Assert(t, strings.Contains(result.Description, scheduledEventCode),
			"Expected description to contain \""+scheduledEventCode+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventStartTime),
			"Expected description to contain \""+scheduledEventStartTime+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventEndTime),
			"Expected description to contain \""+scheduledEventEndTime+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventDescription),
			"Expected description to contain \""+scheduledEventDescription+
				"\"but received \""+result.Description+"\"")

	}()

	err := interruptionevent.MonitorForScheduledEvents(drainChan, cancelChan, imds)
	h.Ok(t, err)
}

func TestMonitorForScheduledEventsMetadataParseFailure(t *testing.T) {
	var requestPath string = ec2metadata.ScheduledEventPath

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if imdsV2TokenPath == req.URL.String() {
			rw.WriteHeader(403)
			return
		}
		h.Equals(t, req.URL.String(), requestPath)
	}))
	defer server.Close()

	drainChan := make(chan interruptionevent.InterruptionEvent)
	cancelChan := make(chan interruptionevent.InterruptionEvent)
	imds := ec2metadata.New("bad url", 0)

	err := interruptionevent.MonitorForScheduledEvents(drainChan, cancelChan, imds)
	h.Assert(t, err != nil, "Failed to return error when metadata parse fails")
}

func TestMonitorForScheduledEvents404Response(t *testing.T) {
	var requestPath string = ec2metadata.ScheduledEventPath

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if imdsV2TokenPath == req.URL.String() {
			rw.WriteHeader(403)
			return
		}
		h.Equals(t, req.URL.String(), requestPath)
		http.Error(rw, "error", http.StatusNotFound)
	}))
	defer server.Close()

	drainChan := make(chan interruptionevent.InterruptionEvent)
	cancelChan := make(chan interruptionevent.InterruptionEvent)
	imds := ec2metadata.New(server.URL, 1)

	err := interruptionevent.MonitorForScheduledEvents(drainChan, cancelChan, imds)
	h.Assert(t, err != nil, "Failed to return error when 404 response")
}

func TestMonitorForScheduledEventsStartTimeParseFail(t *testing.T) {
	var requestPath string = ec2metadata.ScheduledEventPath
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if imdsV2TokenPath == req.URL.String() {
			rw.WriteHeader(403)
			return
		}
		h.Equals(t, req.URL.String(), requestPath)
		rw.Write([]byte(`[{
			"NotBefore": "",
			"Code": "` + scheduledEventCode + `",
			"Description": "` + scheduledEventDescription + `",
			"EventId": "` + scheduledEventId + `",
			"NotAfter": "` + scheduledEventEndTime + `",
			"State": "active"
		}]`))
	}))
	defer server.Close()

	drainChan := make(chan interruptionevent.InterruptionEvent)
	cancelChan := make(chan interruptionevent.InterruptionEvent)
	imds := ec2metadata.New(server.URL, 1)

	err := interruptionevent.MonitorForScheduledEvents(drainChan, cancelChan, imds)
	h.Assert(t, err != nil, "Failed to return error when failed to parse start time")
}

func TestMonitorForScheduledEventsEndTimeParseFail(t *testing.T) {
	var requestPath string = ec2metadata.ScheduledEventPath
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if imdsV2TokenPath == req.URL.String() {
			rw.WriteHeader(403)
			return
		}
		h.Equals(t, req.URL.String(), requestPath)
		rw.Write([]byte(`[{
			"NotBefore": "` + scheduledEventStartTime + `",
			"Code": "` + scheduledEventCode + `",
			"Description": "` + scheduledEventDescription + `",
			"EventId": "` + scheduledEventId + `",
			"NotAfter": "",
			"State": "active"
		}]`))
	}))
	defer server.Close()

	drainChan := make(chan interruptionevent.InterruptionEvent)
	cancelChan := make(chan interruptionevent.InterruptionEvent)
	imds := ec2metadata.New(server.URL, 1)

	go func() {
		result := <-drainChan
		h.Equals(t, scheduledEventId, result.EventID)
		h.Equals(t, interruptionevent.ScheduledEventKind, result.Kind)
		h.Equals(t, scheduledEventState, result.State)
		h.Equals(t, expScheduledEventStartTimeFmt, result.StartTime.String())
		h.Equals(t, expScheduledEventStartTimeFmt, result.EndTime.String())

		h.Assert(t, strings.Contains(result.Description, scheduledEventCode),
			"Expected description to contain \""+scheduledEventCode+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventStartTime),
			"Expected description to contain \""+scheduledEventStartTime+
				"\"but received \""+result.Description+"\"")
		h.Assert(t, strings.Contains(result.Description, scheduledEventDescription),
			"Expected description to contain \""+scheduledEventDescription+
				"\"but received \""+result.Description+"\"")

	}()

	err := interruptionevent.MonitorForScheduledEvents(drainChan, cancelChan, imds)
	h.Ok(t, err)
}
