//nolint:goconst // test schemas contain repetitive standard database field types
package datadiff_test

import (
	"testing"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

func TestTarget_String(t *testing.T) {
	target := datadiff.Target{
		Project: "blm-iris-analytics-prod",
		Dataset: "sales",
		Table:   "fct_sale_lines",
	}

	want := "blm-iris-analytics-prod.sales.fct_sale_lines"
	if got := target.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestTarget_FullyQualified(t *testing.T) {
	target := datadiff.Target{
		Project: "blm-dev-001",
		Dataset: "product",
		Table:   "dim_product",
	}

	want := "`blm-dev-001.product.dim_product`"
	if got := target.FullyQualified(); got != want {
		t.Errorf("FullyQualified() = %q, want %q", got, want)
	}
}

func TestTarget_EmptyFields(t *testing.T) {
	target := datadiff.Target{}
	if got := target.String(); got != ".." {
		t.Errorf("empty target String() = %q, want %q", got, "..")
	}
	if got := target.FullyQualified(); got != "`..`" {
		t.Errorf("empty target FullyQualified() = %q, want %q", got, "`..`")
	}
}

func TestColumn_IsNested(t *testing.T) {
	flat := datadiff.Column{Name: "id", DataType: "INT64"}
	if flat.IsNested() {
		t.Error("flat column should not be nested")
	}

	nested := datadiff.Column{Name: "address", DataType: "RECORD", Children: []datadiff.Column{
		{Name: "city", DataType: "STRING"},
	}}
	if !nested.IsNested() {
		t.Error("record column should be nested")
	}
}

func TestColumn_IsRepeated(t *testing.T) {
	nullable := datadiff.Column{Name: "name", Mode: datadiff.ModeNullable}
	if nullable.IsRepeated() {
		t.Error("nullable column should not be repeated")
	}

	required := datadiff.Column{Name: "id", Mode: datadiff.ModeRequired}
	if required.IsRepeated() {
		t.Error("required column should not be repeated")
	}

	repeated := datadiff.Column{Name: "tags", Mode: datadiff.ModeRepeated}
	if !repeated.IsRepeated() {
		t.Error("repeated column should be repeated")
	}
}

func TestColumn_Flatten_Simple(t *testing.T) {
	col := datadiff.Column{Name: "id", DataType: "INT64", Position: 1}
	flat := col.Flatten("")

	if len(flat) != 1 {
		t.Fatalf("expected 1 column, got %d", len(flat))
	}
	if flat[0].Name != "id" {
		t.Errorf("name = %q, want %q", flat[0].Name, "id")
	}
}

func TestColumn_Flatten_WithPrefix(t *testing.T) {
	col := datadiff.Column{Name: "city", DataType: "STRING"}
	flat := col.Flatten("address")

	if len(flat) != 1 {
		t.Fatalf("expected 1 column, got %d", len(flat))
	}
	if flat[0].Name != "address.city" {
		t.Errorf("name = %q, want %q", flat[0].Name, "address.city")
	}
}

func TestColumn_Flatten_DeeplyNested(t *testing.T) {
	col := datadiff.Column{
		Name:     "location",
		DataType: "RECORD",
		Children: []datadiff.Column{
			{Name: "address", DataType: "RECORD", Children: []datadiff.Column{
				{Name: "street", DataType: "STRING"},
				{Name: "geo", DataType: "RECORD", Children: []datadiff.Column{
					{Name: "lat", DataType: "FLOAT64"},
					{Name: "lng", DataType: "FLOAT64"},
				}},
			}},
			{Name: "name", DataType: "STRING"},
		},
	}

	flat := col.Flatten("")
	expected := []string{
		"location.address.street",
		"location.address.geo.lat",
		"location.address.geo.lng",
		"location.name",
	}

	if len(flat) != len(expected) {
		t.Fatalf("got %d columns, want %d", len(flat), len(expected))
	}
	for i, want := range expected {
		if flat[i].Name != want {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, want)
		}
	}
}

func TestFlattenAll_Mixed(t *testing.T) {
	cols := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "meta", DataType: "RECORD", Children: []datadiff.Column{
			{Name: "created", DataType: "TIMESTAMP"},
		}},
		{Name: "name", DataType: "STRING"},
	}

	flat := datadiff.FlattenAll(cols)
	expected := []string{"id", "meta.created", "name"}

	if len(flat) != len(expected) {
		t.Fatalf("got %d columns, want %d", len(flat), len(expected))
	}
	for i, want := range expected {
		if flat[i].Name != want {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, want)
		}
	}
}

func TestFlattenAll_Empty(t *testing.T) {
	flat := datadiff.FlattenAll(nil)
	if len(flat) != 0 {
		t.Errorf("expected 0 columns, got %d", len(flat))
	}
}

func TestFlattenAll_NestedFields(t *testing.T) {
	cols := []datadiff.Column{
		{Name: "id", DataType: "INT64"},
		{Name: "address", DataType: "RECORD", Children: []datadiff.Column{
			{Name: "street", DataType: "STRING"},
			{Name: "geo", DataType: "RECORD", Children: []datadiff.Column{
				{Name: "lat", DataType: "FLOAT64"},
				{Name: "lng", DataType: "FLOAT64"},
			}},
		}},
		{Name: "name", DataType: "STRING"},
	}

	flat := datadiff.FlattenAll(cols)

	expected := []string{"id", "address.street", "address.geo.lat", "address.geo.lng", "name"}
	if len(flat) != len(expected) {
		t.Fatalf("got %d columns, want %d", len(flat), len(expected))
	}
	for i, want := range expected {
		if flat[i].Name != want {
			t.Errorf("flat[%d].Name = %q, want %q", i, flat[i].Name, want)
		}
	}
}

func TestVolumeResult_String(t *testing.T) {
	match := datadiff.VolumeResult{Match: true, LeftCount: 500, RightCount: 500}
	if s := match.String(); s != "MATCH (500 rows)" {
		t.Errorf("got %q", s)
	}

	mismatch := datadiff.VolumeResult{Match: false, LeftCount: 500, RightCount: 499}
	if s := mismatch.String(); s != "MISMATCH (left=500, right=499, delta=-1)" {
		t.Errorf("got %q", s)
	}
}

func TestResult_Pass(t *testing.T) {
	tests := []struct {
		name   string
		result datadiff.Result
		want   bool
	}{
		{
			name: "all match",
			result: datadiff.Result{
				Schema:  datadiff.SchemaResult{Match: true},
				Volume:  datadiff.VolumeResult{Match: true},
				Content: datadiff.ContentResult{Match: true},
			},
			want: true,
		},
		{
			name: "schema fails",
			result: datadiff.Result{
				Schema:  datadiff.SchemaResult{Match: false},
				Volume:  datadiff.VolumeResult{Match: true},
				Content: datadiff.ContentResult{Match: true},
			},
			want: false,
		},
		{
			name: "volume fails",
			result: datadiff.Result{
				Schema:  datadiff.SchemaResult{Match: true},
				Volume:  datadiff.VolumeResult{Match: false},
				Content: datadiff.ContentResult{Match: true},
			},
			want: false,
		},
		{
			name: "content fails",
			result: datadiff.Result{
				Schema:  datadiff.SchemaResult{Match: true},
				Volume:  datadiff.VolumeResult{Match: true},
				Content: datadiff.ContentResult{Match: false},
			},
			want: false,
		},
		{
			name: "stats fails but pass is true — stats not in gate",
			result: datadiff.Result{
				Schema:  datadiff.SchemaResult{Match: true},
				Volume:  datadiff.VolumeResult{Match: true},
				Content: datadiff.ContentResult{Match: true},
				Stats:   datadiff.StatsResult{Match: false},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Pass(); got != tt.want {
				t.Errorf("Pass() = %v, want %v", got, tt.want)
			}
		})
	}
}
