// SiYuan - https://github.com/siyuan-note/siyuan
// Copyright (c) 2020-present, b3log.org
// SPDX-License-Identifier: AGPL-3.0-or-later

package apicontract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAVContractChartSettingsAndRenderData(t *testing.T) {
	if _, err := RenderAttributeView.Decode(strings.NewReader(`{"id":"database","initialLayout":"chart"}`)); err != nil {
		t.Fatalf("chart initial layout rejected: %v", err)
	}
	bundle, err := BuildBundle()
	if err != nil {
		t.Fatal(err)
	}
	full := `"chartType":"bar","xKeyID":"20261101000000-cstatus","dateBucket":"week","measure":"sum","valueKeyID":"20261101000000-cnumber",` +
		`"stackKeyID":"","sort":"valueDesc","showEmptyGroups":true,"showLegend":true,"showDataLabels":false,"cumulative":false,"height":360`
	for _, fixture := range []struct {
		data  string
		valid bool
	}{
		{`{` + full + `}`, true},
		{`{` + strings.Replace(full, `"cumulative":false`, `"cumulative":true`, 1) + `}`, true},
		{`{` + strings.Replace(full, `"cumulative":false`, `"cumulative":"yes"`, 1) + `}`, false},
		{`{"chartType":"","xKeyID":"","dateBucket":"","measure":"","valueKeyID":"","stackKeyID":"","sort":"","showEmptyGroups":false,"showLegend":false,"showDataLabels":false,"cumulative":false,"height":0}`, true},
		{`{` + strings.Replace(full, `"bar"`, `"pie"`, 1) + `}`, false},
		{`{` + strings.Replace(full, `"week"`, `"hour"`, 1) + `}`, false},
		{`{` + strings.Replace(full, `"sum"`, `"median"`, 1) + `}`, false},
		{`{` + strings.Replace(full, `"valueDesc"`, `"random"`, 1) + `}`, false},
		{`{` + strings.Replace(full, `,"height":360`, ``, 1) + `}`, false},
	} {
		var operation TransactionOperation
		if err = json.Unmarshal([]byte(`{"action":"setAttrViewChart","data":`+fixture.data+`}`), &operation); err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(Success([]*Transaction{{DoOperations: []*TransactionOperation{&operation}}}))
		if err != nil {
			t.Fatal(err)
		}
		if err = bundle.ValidateResponse("POST", "/api/transactions", payload); (err == nil) != fixture.valid {
			t.Fatalf("chart settings %s: valid=%t err=%v", fixture.data, fixture.valid, err)
		}
	}

	table := &AVTable{AVBaseInstance: &AVBaseInstance{ID: "20261101000000-cview01", Filters: []*AVViewFilter{}, Sorts: []*AVViewSort{}},
		Columns: []*AVTableColumn{}, Rows: []*AVTableRow{}, RowCount: 3,
		Chart: &AVChartSettings{ChartType: "donut", Measure: "count", Sort: "default", ShowLegend: true, Height: 360},
		ChartData: &AVChartData{Measure: "count", XType: "status", Categories: []*AVChartCategory{{Key: "Done", Label: "Done", Color: "3", Value: new(2.0), Count: 2}, {Empty: true, Value: new(1.0), Count: 1}},
			Series: []*AVChartSeries{{Values: []*float64{new(2.0), new(1.0)}, Counts: []int{2, 1}, Items: [][]*AVChartItem{{{ID: "item", Title: "Item"}}, {}}}},
			Total:  new(3.0), ItemCount: 3}}
	data, err := json.Marshal(table)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"chart":{"chartType":"donut"`, `"chartData":{"measure":"count"`, `"empty":true`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("chart table payload is missing %s: %s", field, data)
		}
	}
}
