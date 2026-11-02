// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package apicontract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAVContractTimelineCreateRequestAndSettings(t *testing.T) {
	request, err := CreateAttributeViewItem.Decode(strings.NewReader(`{"avID":"av","blockID":"block","viewID":"timeline","calendarDate":1790006400000,"calendarEndDate":1790265600000}`))
	if err != nil || nil == request.CalendarDate || nil == request.CalendarEndDate || *request.CalendarEndDate != 1790265600000 {
		t.Fatalf("timeline end date was not decoded: %+v %v", request, err)
	}
	request, err = CreateAttributeViewItem.Decode(strings.NewReader(`{"avID":"av","blockID":"block","calendarEndDate":null}`))
	if err != nil || nil != request.CalendarEndDate {
		t.Fatalf("null timeline end date must be omitted: %+v %v", request, err)
	}
	for _, value := range []string{`"1790265600000"`, `true`, `1.5`} {
		if _, err = CreateAttributeViewItem.Decode(strings.NewReader(`{"avID":"av","blockID":"block","calendarEndDate":` + value + `}`)); err == nil {
			t.Fatalf("invalid timeline end date accepted: %s", value)
		}
	}
	if _, err = RenderAttributeView.Decode(strings.NewReader(`{"id":"database","initialLayout":"timeline"}`)); err != nil {
		t.Fatalf("timeline initial layout rejected: %v", err)
	}

	bundle, err := BuildBundle()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		data  string
		valid bool
	}{
		{`{"startKeyID":"20261101000000-tstart1","endKeyID":"","zoom":"month","colorBy":"conditional"}`, true},
		{`{"startKeyID":"","endKeyID":"","zoom":"","colorBy":""}`, true},
		{`{"startKeyID":"","endKeyID":"","zoom":"year","colorBy":""}`, false},
		{`{"startKeyID":"","endKeyID":"","zoom":"week"}`, false},
	} {
		var operation TransactionOperation
		if err = json.Unmarshal([]byte(`{"action":"setAttrViewTimeline","data":`+fixture.data+`}`), &operation); err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(Success([]*Transaction{{DoOperations: []*TransactionOperation{&operation}}}))
		if err != nil {
			t.Fatal(err)
		}
		if err = bundle.ValidateResponse("POST", "/api/transactions", payload); (err == nil) != fixture.valid {
			t.Fatalf("timeline settings %s: valid=%t err=%v", fixture.data, fixture.valid, err)
		}
	}
}
