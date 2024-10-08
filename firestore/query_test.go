// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package firestore

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	pb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"cloud.google.com/go/internal/pretty"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestFilterToProto(t *testing.T) {
	for _, test := range []struct {
		in   EntityFilter
		want *pb.StructuredQuery_Filter
	}{
		{
			in: PropertyFilter{"a", ">", 1},
			want: &pb.StructuredQuery_Filter{FilterType: &pb.StructuredQuery_Filter_FieldFilter{
				FieldFilter: &pb.StructuredQuery_FieldFilter{
					Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
					Op:    pb.StructuredQuery_FieldFilter_GREATER_THAN,
					Value: intval(1),
				},
			}},
		},
		{
			in: PropertyFilter{"a", "==", nil},
			want: &pb.StructuredQuery_Filter{FilterType: &pb.StructuredQuery_Filter_UnaryFilter{
				UnaryFilter: &pb.StructuredQuery_UnaryFilter{
					OperandType: &pb.StructuredQuery_UnaryFilter_Field{
						Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
					},
					Op: pb.StructuredQuery_UnaryFilter_IS_NULL,
				},
			}},
		},
		{
			in: PropertyFilter{"a", "==", math.NaN()},
			want: &pb.StructuredQuery_Filter{FilterType: &pb.StructuredQuery_Filter_UnaryFilter{
				UnaryFilter: &pb.StructuredQuery_UnaryFilter{
					OperandType: &pb.StructuredQuery_UnaryFilter_Field{
						Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
					},
					Op: pb.StructuredQuery_UnaryFilter_IS_NAN,
				},
			}},
		},
		{
			in: PropertyPathFilter{[]string{"a"}, ">", 1},
			want: &pb.StructuredQuery_Filter{FilterType: &pb.StructuredQuery_Filter_FieldFilter{
				FieldFilter: &pb.StructuredQuery_FieldFilter{
					Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
					Op:    pb.StructuredQuery_FieldFilter_GREATER_THAN,
					Value: intval(1),
				},
			}},
		},
		{
			in: PropertyPathFilter{[]string{"a"}, "==", nil},
			want: &pb.StructuredQuery_Filter{FilterType: &pb.StructuredQuery_Filter_UnaryFilter{
				UnaryFilter: &pb.StructuredQuery_UnaryFilter{
					OperandType: &pb.StructuredQuery_UnaryFilter_Field{
						Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
					},
					Op: pb.StructuredQuery_UnaryFilter_IS_NULL,
				},
			}},
		},
		{
			in: PropertyPathFilter{[]string{"a"}, "==", math.NaN()},
			want: &pb.StructuredQuery_Filter{FilterType: &pb.StructuredQuery_Filter_UnaryFilter{
				UnaryFilter: &pb.StructuredQuery_UnaryFilter{
					OperandType: &pb.StructuredQuery_UnaryFilter_Field{
						Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
					},
					Op: pb.StructuredQuery_UnaryFilter_IS_NAN,
				},
			}},
		},
		{
			in: OrFilter{
				Filters: []EntityFilter{
					PropertyPathFilter{[]string{"a"}, ">", 5},
					PropertyFilter{"a", "<=", 2},
				},
			},
			want: &pb.StructuredQuery_Filter{
				FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
					CompositeFilter: &pb.StructuredQuery_CompositeFilter{
						Op: pb.StructuredQuery_CompositeFilter_OR,
						Filters: []*pb.StructuredQuery_Filter{
							{
								FilterType: &pb.StructuredQuery_Filter_FieldFilter{
									FieldFilter: &pb.StructuredQuery_FieldFilter{
										Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
										Op:    pb.StructuredQuery_FieldFilter_GREATER_THAN,
										Value: intval(5),
									},
								},
							},
							{
								FilterType: &pb.StructuredQuery_Filter_FieldFilter{
									FieldFilter: &pb.StructuredQuery_FieldFilter{
										Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
										Op:    pb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL,
										Value: intval(2),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			in: AndFilter{
				Filters: []EntityFilter{
					PropertyPathFilter{[]string{"a"}, ">", 5},
					PropertyFilter{"a", "<=", 10},
				},
			},
			want: &pb.StructuredQuery_Filter{
				FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
					CompositeFilter: &pb.StructuredQuery_CompositeFilter{
						Op: pb.StructuredQuery_CompositeFilter_AND,
						Filters: []*pb.StructuredQuery_Filter{
							{
								FilterType: &pb.StructuredQuery_Filter_FieldFilter{
									FieldFilter: &pb.StructuredQuery_FieldFilter{
										Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
										Op:    pb.StructuredQuery_FieldFilter_GREATER_THAN,
										Value: intval(5),
									},
								},
							},
							{
								FilterType: &pb.StructuredQuery_Filter_FieldFilter{
									FieldFilter: &pb.StructuredQuery_FieldFilter{
										Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
										Op:    pb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL,
										Value: intval(10),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			in: OrFilter{
				Filters: []EntityFilter{
					PropertyPathFilter{[]string{"b"}, "==", 15},
					AndFilter{
						Filters: []EntityFilter{
							PropertyPathFilter{[]string{"a"}, ">", 5},
							PropertyFilter{"a", "<=", 12},
						},
					},
				},
			},
			want: &pb.StructuredQuery_Filter{
				FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
					CompositeFilter: &pb.StructuredQuery_CompositeFilter{
						Op: pb.StructuredQuery_CompositeFilter_OR,
						Filters: []*pb.StructuredQuery_Filter{
							{
								FilterType: &pb.StructuredQuery_Filter_FieldFilter{
									FieldFilter: &pb.StructuredQuery_FieldFilter{
										Field: &pb.StructuredQuery_FieldReference{FieldPath: "b"},
										Op:    pb.StructuredQuery_FieldFilter_EQUAL,
										Value: intval(15),
									},
								},
							},
							{
								FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
									CompositeFilter: &pb.StructuredQuery_CompositeFilter{
										Op: pb.StructuredQuery_CompositeFilter_AND,
										Filters: []*pb.StructuredQuery_Filter{
											{
												FilterType: &pb.StructuredQuery_Filter_FieldFilter{
													FieldFilter: &pb.StructuredQuery_FieldFilter{
														Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
														Op:    pb.StructuredQuery_FieldFilter_GREATER_THAN,
														Value: intval(5),
													},
												},
											},
											{
												FilterType: &pb.StructuredQuery_Filter_FieldFilter{
													FieldFilter: &pb.StructuredQuery_FieldFilter{
														Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
														Op:    pb.StructuredQuery_FieldFilter_LESS_THAN_OR_EQUAL,
														Value: intval(12),
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	} {
		got, err := test.in.toProto()
		if err != nil {
			t.Fatal(err)
		}
		if !testEqual(got, test.want) {
			t.Errorf("%+v:\ngot\n%v\nwant\n%v", test.in, pretty.Value(got), pretty.Value(test.want))
		}
	}
}

type toProtoScenario struct {
	desc string
	in   Query
	want *pb.RunQueryRequest
}

// Creates protos used to test toProto, FromProto, ToProto funcs.
func createTestScenarios(t *testing.T) []toProtoScenario {
	filtr := func(path []string, op string, val interface{}) *pb.StructuredQuery_Filter {
		f, err := PropertyPathFilter{path, op, val}.toProto()
		if err != nil {
			t.Fatal(err)
		}
		return f
	}

	c := &Client{projectID: "P", databaseID: "DB"}
	coll := c.Collection("C")
	q := coll.Query
	docsnap := &DocumentSnapshot{
		Ref: coll.Doc("D"),
		proto: &pb.Document{
			Fields: map[string]*pb.Value{"a": intval(7), "b": intval(8), "c": arrayval(intval(1), intval(2))},
		},
	}

	return []toProtoScenario{
		{
			desc: "q.Select()",
			in:   q.Select(),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Select: &pb.StructuredQuery_Projection{
							Fields: []*pb.StructuredQuery_FieldReference{fref1("__name__")},
						},
					},
				},
			},
		},
		{
			desc: `q.Select("a", "b")`,
			in:   q.Select("a", "b"),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Select: &pb.StructuredQuery_Projection{
							Fields: []*pb.StructuredQuery_FieldReference{fref1("a"), fref1("b")},
						},
					},
				},
			},
		},
		{
			desc: `q.Select("a", "b").Select("c")`,
			in:   q.Select("a", "b").Select("c"), // last wins
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Select: &pb.StructuredQuery_Projection{
							Fields: []*pb.StructuredQuery_FieldReference{fref1("c")},
						},
					},
				},
			},
		},
		{
			desc: `q.SelectPaths([]string{"*"}, []string{"/"})`,
			in:   q.SelectPaths([]string{"*"}, []string{"/"}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Select: &pb.StructuredQuery_Projection{
							Fields: []*pb.StructuredQuery_FieldReference{fref1("*"), fref1("/")},
						},
					},
				},
			},
		},
		{
			desc: `q.Where("a", ">", 5)`,
			in:   q.Where("a", ">", 5),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"a"}, ">", 5)},
				},
			},
		},
		{
			desc: `q.Where("a", "==", NaN)`,
			in:   q.Where("a", "==", float32(math.NaN())),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"a"}, "==", math.NaN())},
				},
			},
		},
		{
			desc: `q.Where("a", "!=", 3)`,
			in:   q.Where("a", "!=", 3),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"a"}, "!=", 3)},
				},
			},
		},
		{
			desc: `q.Where("a", "in", []int{7, 8})`,
			in:   q.Where("a", "in", []int{7, 8}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"a"}, "in", []int{7, 8})},
				},
			},
		},
		{
			desc: `q.Where("a", "not-in", []int{9})`,
			in:   q.Where("a", "not-in", []int{9}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"a"}, "not-in", []int{9})},
				},
			},
		},
		{
			desc: `q.Where("c", "array-contains", 1)`,
			in:   q.Where("c", "array-contains", 1),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"c"}, "array-contains", 1)},
				},
			},
		},
		{
			desc: `q.Where("c", "array-contains-any", []int{1, 2})`,
			in:   q.Where("c", "array-contains-any", []int{1, 2}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"c"}, "array-contains-any", []int{1, 2})},
				},
			},
		},
		{
			desc: `q.Where("c", "array-contains-any", []int{1, 2}).RunOptions(ExplainOptions{Analyze: true})`,
			in:   q.Where("c", "array-contains-any", []int{1, 2}).WithRunOptions(ExplainOptions{Analyze: true}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"c"}, "array-contains-any", []int{1, 2})},
				},
				ExplainOptions: &pb.ExplainOptions{
					Analyze: true,
				},
			},
		},
		{
			desc: `q.Where("c", "array-contains-any", []int{1, 2}).RunOptions(ExplainOptions{Analyze: false})`,
			in:   q.Where("c", "array-contains-any", []int{1, 2}).WithRunOptions(ExplainOptions{Analyze: false}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"c"}, "array-contains-any", []int{1, 2})},
				},
				ExplainOptions: &pb.ExplainOptions{
					Analyze: false,
				},
			},
		},
		{
			desc: `q.Where("c", "array-contains-any", []int{1, 2}) RunOptions invoked multiple times`,
			in: q.Where("c", "array-contains-any", []int{1, 2}).
				WithRunOptions(ExplainOptions{Analyze: false}).
				WithRunOptions(ExplainOptions{Analyze: true}),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"c"}, "array-contains-any", []int{1, 2})},
				},
				ExplainOptions: &pb.ExplainOptions{
					Analyze: true,
				},
			},
		},
		{
			desc: `q.Where("a", ">", 5).Where("b", "<", "foo")`,
			in:   q.Where("a", ">", 5).Where("b", "<", "foo"),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: &pb.StructuredQuery_Filter{
							FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
								CompositeFilter: &pb.StructuredQuery_CompositeFilter{
									Op: pb.StructuredQuery_CompositeFilter_AND,
									Filters: []*pb.StructuredQuery_Filter{
										filtr([]string{"a"}, ">", 5), filtr([]string{"b"}, "<", "foo"),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: `q.WhereEntity(AndFilter({"a", ">", 5}, {"b", "<", "foo"}))`,
			in: q.WhereEntity(
				AndFilter{
					Filters: []EntityFilter{
						PropertyFilter{
							Path:     "a",
							Operator: ">",
							Value:    5,
						},
						PropertyFilter{
							Path:     "b",
							Operator: "<",
							Value:    "foo",
						},
					},
				},
			),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,
				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: &pb.StructuredQuery_Filter{
							FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
								CompositeFilter: &pb.StructuredQuery_CompositeFilter{
									Op: pb.StructuredQuery_CompositeFilter_AND,
									Filters: []*pb.StructuredQuery_Filter{
										{
											FilterType: &pb.StructuredQuery_Filter_FieldFilter{
												FieldFilter: &pb.StructuredQuery_FieldFilter{
													Field: &pb.StructuredQuery_FieldReference{FieldPath: "a"},
													Op:    pb.StructuredQuery_FieldFilter_GREATER_THAN,
													Value: intval(5),
												},
											},
										},
										{
											FilterType: &pb.StructuredQuery_Filter_FieldFilter{
												FieldFilter: &pb.StructuredQuery_FieldFilter{
													Field: &pb.StructuredQuery_FieldReference{FieldPath: "b"},
													Op:    pb.StructuredQuery_FieldFilter_LESS_THAN,
													Value: strval("foo"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			desc: `  q.WherePath([]string{"/", "*"}, ">", 5)`,
			in:   q.WherePath([]string{"/", "*"}, ">", 5),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{Where: filtr([]string{"/", "*"}, ">", 5)},
				},
			},
		},
		{
			desc: `q.OrderBy("b", Asc).OrderBy("a", Desc).OrderByPath([]string{"~"}, Asc)`,
			in:   q.OrderBy("b", Asc).OrderBy("a", Desc).OrderByPath([]string{"~"}, Asc),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("b"), Direction: pb.StructuredQuery_ASCENDING},
							{Field: fref1("a"), Direction: pb.StructuredQuery_DESCENDING},
							{Field: fref1("~"), Direction: pb.StructuredQuery_ASCENDING},
						},
					},
				},
			},
		},
		{
			desc: `q.Offset(2).Limit(3)`,
			in:   q.Offset(2).Limit(3),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Offset: 2,
						Limit:  &wrapperspb.Int32Value{Value: 3},
					},
				},
			},
		},
		{
			desc: `q.Offset(2).Limit(3).Limit(4).Offset(5)`,
			in:   q.Offset(2).Limit(3).Limit(4).Offset(5), // last wins
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Offset: 5,
						Limit:  &wrapperspb.Int32Value{Value: 4},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Asc).StartAt(7).EndBefore(9)`,
			in:   q.OrderBy("a", Asc).StartAt(7).EndBefore(9),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7)},
							Before: true,
						},
						EndAt: &pb.Cursor{
							Values: []*pb.Value{intval(9)},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Asc).StartAt(7).EndAt(9)`,
			in:   q.OrderBy("a", Asc).StartAt(7).EndAt(9),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7)},
							Before: true,
						},
						EndAt: &pb.Cursor{
							Values: []*pb.Value{intval(9)},
							Before: false,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Asc).StartAfter(7).EndAt(9)`,
			in:   q.OrderBy("a", Asc).StartAfter(7).EndAt(9),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7)},
							Before: false,
						},
						EndAt: &pb.Cursor{
							Values: []*pb.Value{intval(9)},
							Before: false,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy(DocumentID, Asc).StartAfter("foo").EndBefore("bar")`,
			in:   q.OrderBy(DocumentID, Asc).StartAfter("foo").EndBefore("bar"),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{refval(coll.parentPath + "/C/foo")},
							Before: false,
						},
						EndAt: &pb.Cursor{
							Values: []*pb.Value{refval(coll.parentPath + "/C/bar")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Asc).OrderBy("b", Desc).StartAfter(7, 8).EndAt(9, 10)`,
			in:   q.OrderBy("a", Asc).OrderBy("b", Desc).StartAfter(7, 8).EndAt(9, 10),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
							{Field: fref1("b"), Direction: pb.StructuredQuery_DESCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7), intval(8)},
							Before: false,
						},
						EndAt: &pb.Cursor{
							Values: []*pb.Value{intval(9), intval(10)},
							Before: false,
						},
					},
				},
			},
		},
		{
			// last of StartAt/After wins, same for End
			desc: `q.OrderBy("a", Asc).StartAfter(1).StartAt(2).EndAt(3).EndBefore(4)`,
			in: q.OrderBy("a", Asc).
				StartAfter(1).StartAt(2).
				EndAt(3).EndBefore(4),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(2)},
							Before: true,
						},
						EndAt: &pb.Cursor{
							Values: []*pb.Value{intval(4)},
							Before: true,
						},
					},
				},
			},
		},
		// Start/End with DocumentSnapshot
		// These tests are from the "Document Snapshot Cursors" doc.
		{
			desc: `q.StartAt(docsnap)`,
			in:   q.StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Asc).StartAt(docsnap)`,
			in:   q.OrderBy("a", Asc).StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7), refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Desc).StartAt(docsnap)`,
			in:   q.OrderBy("a", Desc).StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_DESCENDING},
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_DESCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7), refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.OrderBy("a", Desc).OrderBy("b", Asc).StartAt(docsnap)`,
			in:   q.OrderBy("a", Desc).OrderBy("b", Asc).StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_DESCENDING},
							{Field: fref1("b"), Direction: pb.StructuredQuery_ASCENDING},
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7), intval(8), refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.Where("a", "==", 3).StartAt(docsnap)`,
			in:   q.Where("a", "==", 3).StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: filtr([]string{"a"}, "==", 3),
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.Where("a", "<", 3).StartAt(docsnap)`,
			in:   q.Where("a", "<", 3).StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: filtr([]string{"a"}, "<", 3),
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7), refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.Where("b", "==", 1).Where("a", "<", 3).StartAt(docsnap)`,
			in:   q.Where("b", "==", 1).Where("a", "<", 3).StartAt(docsnap),
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: &pb.StructuredQuery_Filter{
							FilterType: &pb.StructuredQuery_Filter_CompositeFilter{
								CompositeFilter: &pb.StructuredQuery_CompositeFilter{
									Op: pb.StructuredQuery_CompositeFilter_AND,
									Filters: []*pb.StructuredQuery_Filter{
										filtr([]string{"b"}, "==", 1),
										filtr([]string{"a"}, "<", 3),
									},
								},
							},
						},
						OrderBy: []*pb.StructuredQuery_Order{
							{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
							{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
						},
						StartAt: &pb.Cursor{
							Values: []*pb.Value{intval(7), refval(coll.parentPath + "/C/D")},
							Before: true,
						},
					},
				},
			},
		},
		{
			desc: `q.Where("a", ">", 5).FindNearest float64 vector`,
			in: q.Where("a", ">", 5).
				FindNearest("embeddedField", []float64{100, 200, 300}, 2, DistanceMeasureEuclidean, nil).q,
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: filtr([]string{"a"}, ">", 5),
						FindNearest: &pb.StructuredQuery_FindNearest{
							VectorField: fref1("embeddedField"),
							QueryVector: &pb.Value{
								ValueType: &pb.Value_MapValue{
									MapValue: &pb.MapValue{
										Fields: map[string]*pb.Value{
											typeKey: stringToProtoValue(typeValVector),
											valueKey: {
												ValueType: &pb.Value_ArrayValue{
													ArrayValue: &pb.ArrayValue{
														Values: []*pb.Value{
															{ValueType: &pb.Value_DoubleValue{DoubleValue: 100}},
															{ValueType: &pb.Value_DoubleValue{DoubleValue: 200}},
															{ValueType: &pb.Value_DoubleValue{DoubleValue: 300}},
														},
													},
												},
											},
										},
									},
								},
							},
							Limit:           &wrapperspb.Int32Value{Value: trunc32(2)},
							DistanceMeasure: pb.StructuredQuery_FindNearest_EUCLIDEAN,
						},
					},
				},
			},
		},
		{
			desc: `q.Where("a", ">", 5).FindNearest float32 vector`,
			in: q.Where("a", ">", 5).
				FindNearest("embeddedField", []float32{100, 200, 300}, 2, DistanceMeasureEuclidean, nil).q,
			want: &pb.RunQueryRequest{
				Parent: q.parentPath,

				QueryType: &pb.RunQueryRequest_StructuredQuery{
					StructuredQuery: &pb.StructuredQuery{
						Where: filtr([]string{"a"}, ">", 5),
						FindNearest: &pb.StructuredQuery_FindNearest{
							VectorField: fref1("embeddedField"),
							QueryVector: &pb.Value{
								ValueType: &pb.Value_MapValue{
									MapValue: &pb.MapValue{
										Fields: map[string]*pb.Value{
											typeKey: stringToProtoValue(typeValVector),
											valueKey: {
												ValueType: &pb.Value_ArrayValue{
													ArrayValue: &pb.ArrayValue{
														Values: []*pb.Value{
															{ValueType: &pb.Value_DoubleValue{DoubleValue: 100}},
															{ValueType: &pb.Value_DoubleValue{DoubleValue: 200}},
															{ValueType: &pb.Value_DoubleValue{DoubleValue: 300}},
														},
													},
												},
											},
										},
									},
								},
							},
							Limit:           &wrapperspb.Int32Value{Value: trunc32(2)},
							DistanceMeasure: pb.StructuredQuery_FindNearest_EUCLIDEAN,
						},
					},
				},
			},
		},
	}
}

func TestQueryToProto(t *testing.T) {
	for _, test := range createTestScenarios(t) {
		got, err := test.in.toRunQueryRequestProto()
		if err != nil {
			t.Fatalf("%s: %v", test.desc, err)
		}
		pbStructuredQuery := test.want.QueryType.(*pb.RunQueryRequest_StructuredQuery)
		pbStructuredQuery.StructuredQuery.From = []*pb.StructuredQuery_CollectionSelector{{CollectionId: "C"}}
		if !testEqual(got, test.want) {

			t.Fatalf("%s:\ngot\n%v\nwant\n%v\ndiff\n%v", test.desc, pretty.Value(got), pretty.Value(test.want), testDiff(got, test.want))
		}
	}
}

// Convert a Query to a Proto and back again verifying roundtripping.
// We cannot in general verify the round trip from Query back to Query,
// because information is lost. But we can check that the deserialized query's proto
// matches the original proto.
func TestQueryFromProtoRoundTrip(t *testing.T) {
	c := &Client{projectID: "P", databaseID: "DB"}
	for _, test := range createTestScenarios(t) {
		t.Run(test.desc, func(t *testing.T) {
			protoBytes, err := test.in.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			gotq, err := Query{c: c}.Deserialize(protoBytes)
			if err != nil {
				t.Fatal(err)
			}
			got, err := gotq.toRunQueryRequestProto()
			want := test.want
			want.QueryType.(*pb.RunQueryRequest_StructuredQuery).StructuredQuery.From = []*pb.StructuredQuery_CollectionSelector{{CollectionId: "C"}}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("mismatch (-want, +got)\n: %s", diff)
			}
		})
	}
}

func fref1(s string) *pb.StructuredQuery_FieldReference {
	ref, _ := fref([]string{s})
	return ref
}

func TestQueryToProtoErrors(t *testing.T) {
	st := map[string]interface{}{"a": ServerTimestamp}
	del := map[string]interface{}{"a": Delete}
	c := &Client{projectID: "P", databaseID: "DB"}
	coll := c.Collection("C")
	docsnap := &DocumentSnapshot{
		Ref: coll.Doc("D"),
		proto: &pb.Document{
			Fields: map[string]*pb.Value{"a": intval(7)},
		},
	}
	q := coll.Query
	for i, query := range []Query{
		{},                                     // no collection ID
		q.Where("x", "<>", 1),                  // invalid operator
		q.Where("~", ">", 1),                   // invalid path
		q.WherePath([]string{"*", ""}, ">", 1), // invalid path
		q.WhereEntity( // invalid nested filters
			AndFilter{
				Filters: []EntityFilter{
					PropertyFilter{
						Path:     "x",
						Operator: "<>",
						Value:    1,
					},
					PropertyFilter{
						Path:     "~",
						Operator: ">",
						Value:    1,
					},
				},
			},
		),
		q.StartAt(1), // no OrderBy
		q.StartAt(2).OrderBy("x", Asc).OrderBy("y", Desc), // wrong # OrderBy
		q.Select("*"),                         // invalid path
		q.SelectPaths([]string{"/", "", "~"}), // invalid path
		q.OrderBy("[", Asc),                   // invalid path
		q.OrderByPath([]string{""}, Desc),     // invalid path
		q.Where("x", "==", st),                // ServerTimestamp in filter
		q.OrderBy("a", Asc).StartAt(st),       // ServerTimestamp in Start
		q.OrderBy("a", Asc).EndAt(st),         // ServerTimestamp in End
		q.Where("x", "==", del),               // Delete in filter
		q.OrderBy("a", Asc).StartAt(del),      // Delete in Start
		q.OrderBy("a", Asc).EndAt(del),        // Delete in End
		q.OrderBy(DocumentID, Asc).StartAt(7), // wrong type for __name__
		q.OrderBy(DocumentID, Asc).EndAt(7),   // wrong type for __name__
		q.OrderBy("b", Asc).StartAt(docsnap),  // doc snapshot does not have order-by field
		q.StartAt(docsnap).EndAt("x"),         // mixed doc snapshot and fields
		q.StartAfter("x").EndBefore(docsnap),  // mixed doc snapshot and fields
	} {
		_, err := query.toProto()
		if err == nil {
			t.Errorf("query %d \"%+v\": got nil, want error", i, query)
		}
	}
}

func TestQueryMethodsDoNotModifyReceiver(t *testing.T) {
	var empty Query

	q := Query{}
	_ = q.Select("a", "b")
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}

	q = Query{}
	q1 := q.Where("a", ">", 3)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}
	// Extra check because Where appends to a slice.
	q1before := q.Where("a", ">", 3) // same as q1
	_ = q1.Where("b", "<", "foo")
	if !testEqual(q1, q1before) {
		t.Errorf("got %+v, want %+v", q1, q1before)
	}

	q = Query{}
	q1 = q.OrderBy("a", Asc)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}
	// Extra check because Where appends to a slice.
	q1before = q.OrderBy("a", Asc) // same as q1
	_ = q1.OrderBy("b", Desc)
	if !testEqual(q1, q1before) {
		t.Errorf("got %+v, want %+v", q1, q1before)
	}

	q = Query{}
	_ = q.Offset(5)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}

	q = Query{}
	_ = q.Limit(5)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}

	q = Query{}
	_ = q.StartAt(5)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}

	q = Query{}
	_ = q.StartAfter(5)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}

	q = Query{}
	_ = q.EndAt(5)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}

	q = Query{}
	_ = q.EndBefore(5)
	if !testEqual(q, empty) {
		t.Errorf("got %+v, want empty", q)
	}
}

func TestQueryFromCollectionRef(t *testing.T) {
	c := &Client{projectID: "P", databaseID: "D"}
	coll := c.Collection("C")
	got := coll.Select("x").Offset(8)
	ref, _ := fref(FieldPath{"x"})
	want := Query{
		c:            c,
		parentPath:   c.path() + "/documents",
		path:         "projects/P/databases/D/documents/C",
		collectionID: "C",
		selection: []*pb.StructuredQuery_FieldReference{
			ref,
		},
		offset: 8,
	}
	if !testEqual(got, want) {
		t.Fatalf("\ngot  %+v, \nwant %+v", got, want)
	}
}

func TestQueryGetAll(t *testing.T) {
	// This implicitly tests DocumentIterator as well.
	const dbPath = "projects/projectID/databases/(default)"
	ctx := context.Background()
	c, srv, cleanup := newMock(t)
	srv.reset()
	defer cleanup()

	docNames := []string{"C/a", "C/b"}
	wantPBDocs := []*pb.Document{
		{
			Name:       dbPath + "/documents/" + docNames[0],
			CreateTime: aTimestamp,
			UpdateTime: aTimestamp,
			Fields:     map[string]*pb.Value{"f": intval(2)},
		},
		{
			Name:       dbPath + "/documents/" + docNames[1],
			CreateTime: aTimestamp2,
			UpdateTime: aTimestamp3,
			Fields:     map[string]*pb.Value{"f": intval(1)},
		},
	}
	wantReadTimes := []*tspb.Timestamp{aTimestamp, aTimestamp2}
	srv.addRPC(nil, []interface{}{
		&pb.RunQueryResponse{Document: wantPBDocs[0], ReadTime: aTimestamp},
		&pb.RunQueryResponse{Document: wantPBDocs[1], ReadTime: aTimestamp2},
	})
	gotDocs, err := c.Collection("C").Documents(ctx).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("gotDocs: %+v\n", gotDocs)
	if got, want := len(gotDocs), len(wantPBDocs); got != want {
		t.Errorf("got %d docs, wanted %d", got, want)
	}
	for i, got := range gotDocs {
		want, err := newDocumentSnapshot(c.Doc(docNames[i]), wantPBDocs[i], c, wantReadTimes[i])
		if err != nil {
			t.Fatal(err)
		}
		if !testEqual(got, want) {
			// avoid writing a cycle
			got.c = nil
			want.c = nil
			t.Errorf("#%d: got %+v, want %+v", i, pretty.Value(got), pretty.Value(want))
		}
	}
}

func TestQueryCompareFunc(t *testing.T) {
	mv := func(fields ...interface{}) map[string]*pb.Value {
		m := map[string]*pb.Value{}
		for i := 0; i < len(fields); i += 2 {
			m[fields[i].(string)] = fields[i+1].(*pb.Value)
		}
		return m
	}
	snap := func(ref *DocumentRef, fields map[string]*pb.Value) *DocumentSnapshot {
		return &DocumentSnapshot{Ref: ref, proto: &pb.Document{Fields: fields}}
	}

	c := &Client{}
	coll := c.Collection("C")
	doc1 := coll.Doc("doc1")
	doc2 := coll.Doc("doc2")
	doc3 := coll.Doc("doc3")
	doc4 := coll.Doc("doc4")
	for _, test := range []struct {
		q    Query
		in   []*DocumentSnapshot
		want []*DocumentSnapshot
	}{
		{
			q: coll.OrderBy("foo", Asc),
			in: []*DocumentSnapshot{
				snap(doc3, mv("foo", intval(2))),
				snap(doc4, mv("foo", intval(1))),
				snap(doc2, mv("foo", intval(2))),
			},
			want: []*DocumentSnapshot{
				snap(doc4, mv("foo", intval(1))),
				snap(doc2, mv("foo", intval(2))),
				snap(doc3, mv("foo", intval(2))),
			},
		},
		{
			q: coll.OrderBy("foo", Desc),
			in: []*DocumentSnapshot{
				snap(doc3, mv("foo", intval(2))),
				snap(doc4, mv("foo", intval(1))),
				snap(doc2, mv("foo", intval(2))),
			},
			want: []*DocumentSnapshot{
				snap(doc3, mv("foo", intval(2))),
				snap(doc2, mv("foo", intval(2))),
				snap(doc4, mv("foo", intval(1))),
			},
		},
		{
			q: coll.OrderBy("foo.bar", Asc),
			in: []*DocumentSnapshot{
				snap(doc1, mv("foo", mapval(mv("bar", intval(1))))),
				snap(doc2, mv("foo", mapval(mv("bar", intval(2))))),
				snap(doc3, mv("foo", mapval(mv("bar", intval(2))))),
			},
			want: []*DocumentSnapshot{
				snap(doc1, mv("foo", mapval(mv("bar", intval(1))))),
				snap(doc2, mv("foo", mapval(mv("bar", intval(2))))),
				snap(doc3, mv("foo", mapval(mv("bar", intval(2))))),
			},
		},
		{
			q: coll.OrderBy("foo.bar", Desc),
			in: []*DocumentSnapshot{
				snap(doc1, mv("foo", mapval(mv("bar", intval(1))))),
				snap(doc2, mv("foo", mapval(mv("bar", intval(2))))),
				snap(doc3, mv("foo", mapval(mv("bar", intval(2))))),
			},
			want: []*DocumentSnapshot{
				snap(doc3, mv("foo", mapval(mv("bar", intval(2))))),
				snap(doc2, mv("foo", mapval(mv("bar", intval(2))))),
				snap(doc1, mv("foo", mapval(mv("bar", intval(1))))),
			},
		},
	} {
		got := append([]*DocumentSnapshot(nil), test.in...)
		sort.Sort(byQuery{test.q.compareFunc(), got})
		if diff := testDiff(got, test.want); diff != "" {
			t.Errorf("%+v: %s", test.q, diff)
		}
	}

	// Want error on missing field.
	q := coll.OrderBy("bar", Asc)
	if q.err != nil {
		t.Fatalf("bad query: %v", q.err)
	}
	cf := q.compareFunc()
	s := snap(doc1, mv("foo", intval(1)))
	if _, err := cf(s, s); err == nil {
		t.Error("got nil, want error")
	}
}

func TestQuerySubCollections(t *testing.T) {
	c := &Client{projectID: "P", databaseID: "DB"}

	/*
		        parent-collection
			+---------+  +---------+
			|                      |
			|                      |
		parent-doc			some-other-parent-doc
			|
			|
		sub-collection
			|
			|
		sub-doc
			|
			|
		sub-sub-collection
			|
			|
		sub-sub-doc
	*/
	parentColl := c.Collection("parent-collection")
	parentDoc := parentColl.Doc("parent-doc")
	someOtherParentDoc := parentColl.Doc("some-other-parent-doc")
	subColl := parentDoc.Collection("sub-collection")
	subDoc := subColl.Doc("sub-doc")
	subSubColl := subDoc.Collection("sub-sub-collection")
	subSubDoc := subSubColl.Doc("sub-sub-doc")
	collGroup := c.CollectionGroup("collection-group")

	testCases := []struct {
		queryColl      *Query
		queryFilterDoc *DocumentRef // startAt or endBefore
		wantColl       string
		wantRef        string
		wantErr        bool
	}{
		// Queries are allowed at depth 0.
		{parentColl.query(), parentDoc, "parent-collection", "projects/P/databases/DB/documents/parent-collection/parent-doc", false},
		// Queries are allowed at any depth.
		{subColl.query(), subDoc, "sub-collection", "projects/P/databases/DB/documents/parent-collection/parent-doc/sub-collection/sub-doc", false},
		// Queries must be on immediate children (not allowed on grandchildren),
		// except for CollectionGroup queries.
		{subColl.query(), someOtherParentDoc, "", "", true},
		{collGroup.query(), someOtherParentDoc, "collection-group", "projects/P/databases/DB/documents/parent-collection/some-other-parent-doc", false},
		// Queries must be on immediate children (not allowed on siblings).
		{subColl.query(), subSubDoc, "", "", true},
	}

	// startAt
	for _, testCase := range testCases {
		// Query a child within the document.
		q := testCase.queryColl.StartAt(&DocumentSnapshot{
			Ref: testCase.queryFilterDoc,
			proto: &pb.Document{
				Fields: map[string]*pb.Value{"a": intval(7)},
			},
		}).OrderBy("a", Asc)
		got, err := q.toProto()
		if testCase.wantErr {
			if err == nil {
				t.Fatal("expected err, got nil")
			}
			continue
		}

		if err != nil {
			t.Fatal(err)
		}
		want := &pb.StructuredQuery{
			From: []*pb.StructuredQuery_CollectionSelector{
				{CollectionId: testCase.wantColl, AllDescendants: testCase.queryColl.allDescendants},
			},
			OrderBy: []*pb.StructuredQuery_Order{
				{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
				{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
			},
			StartAt: &pb.Cursor{
				Values: []*pb.Value{
					intval(7),
					// This is the only part of the assertion we really care about.
					refval(testCase.wantRef),
				},
				Before: true,
			},
		}
		if !testEqual(got, want) {
			t.Fatalf("got\n%v\nwant\n%v", pretty.Value(got), pretty.Value(want))
		}
	}

	// endBefore
	for _, testCase := range testCases {
		// Query a child within the document.
		q := testCase.queryColl.EndBefore(&DocumentSnapshot{
			Ref: testCase.queryFilterDoc,
			proto: &pb.Document{
				Fields: map[string]*pb.Value{"a": intval(7)},
			},
		}).OrderBy("a", Asc)
		got, err := q.toProto()
		if testCase.wantErr {
			if err == nil {
				t.Fatal("expected err, got nil")
			}
			continue
		}

		if err != nil {
			t.Fatal(err)
		}
		want := &pb.StructuredQuery{
			From: []*pb.StructuredQuery_CollectionSelector{
				{CollectionId: testCase.wantColl, AllDescendants: testCase.queryColl.allDescendants},
			},
			OrderBy: []*pb.StructuredQuery_Order{
				{Field: fref1("a"), Direction: pb.StructuredQuery_ASCENDING},
				{Field: fref1("__name__"), Direction: pb.StructuredQuery_ASCENDING},
			},
			EndAt: &pb.Cursor{
				Values: []*pb.Value{
					intval(7),
					// This is the only part of the assertion we really care about.
					refval(testCase.wantRef),
				},
				Before: true,
			},
		}
		if !testEqual(got, want) {
			t.Fatalf("got\n%v\nwant\n%v", pretty.Value(got), pretty.Value(want))
		}
	}
}

// Stop should be callable on an uninitialized QuerySnapshotIterator.
func TestStop_Uninitialized(t *testing.T) {
	i := &QuerySnapshotIterator{}
	i.Stop()
}

type byQuery struct {
	compare func(d1, d2 *DocumentSnapshot) (int, error)
	docs    []*DocumentSnapshot
}

func (b byQuery) Len() int      { return len(b.docs) }
func (b byQuery) Swap(i, j int) { b.docs[i], b.docs[j] = b.docs[j], b.docs[i] }
func (b byQuery) Less(i, j int) bool {
	c, err := b.compare(b.docs[i], b.docs[j])
	if err != nil {
		panic(err)
	}
	return c < 0
}

func TestAggregationQuery(t *testing.T) {
	ctx := context.Background()
	c, srv, cleanup := newMock(t)
	defer cleanup()

	srv.addRPC(nil, []interface{}{
		&pb.RunAggregationQueryResponse{
			Result: &pb.AggregationResult{
				AggregateFields: map[string]*pb.Value{
					"testAlias": intval(1),
				},
			},
		},
	})

	q := c.Collection("coll1").Where("f", "==", 2)
	ar, err := q.NewAggregationQuery().WithCount("testAlias").Get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	count, ok := ar["testAlias"]
	if !ok {
		t.Errorf("aggregation query key not found")
	}

	cv := count.(*pb.Value)
	if cv.GetIntegerValue() != 1 {
		t.Errorf("got: %v\nwant: %v\n; result: %v\n", cv.GetIntegerValue(), 1, count)
	}
}

func TestWithSum(t *testing.T) {
	ctx := context.Background()
	sumAlias := "sum"
	c, srv, cleanup := newMock(t)
	defer cleanup()

	srv.addRPC(nil, []interface{}{
		&pb.RunAggregationQueryResponse{
			Result: &pb.AggregationResult{
				AggregateFields: map[string]*pb.Value{
					"sum": intval(1),
				},
			},
		},
	})

	testcases := []struct {
		desc    string
		path    string
		wantErr bool
	}{
		{
			desc:    "Invalid path",
			path:    "path*",
			wantErr: true,
		},
		{
			desc:    "Valid path",
			path:    "path",
			wantErr: false,
		},
	}
	for _, tc := range testcases {

		query := c.Collection("C")
		aggQuery := query.NewAggregationQuery().WithSum(tc.path, sumAlias)
		_, err := aggQuery.Get(ctx)
		if err == nil && tc.wantErr {
			t.Fatalf("%s: got nil wanted error", tc.desc)
		} else if err != nil && !tc.wantErr {
			t.Fatalf("%s: got %v, want nil", tc.desc, err)
		}
	}
}

func TestWithSumPath(t *testing.T) {
	ctx := context.Background()
	sumAlias := "sum"
	c, srv, cleanup := newMock(t)
	defer cleanup()

	srv.addRPC(nil, []interface{}{
		&pb.RunAggregationQueryResponse{
			Result: &pb.AggregationResult{
				AggregateFields: map[string]*pb.Value{
					"sum": intval(1),
				},
			},
		},
	})

	testcases := []struct {
		desc      string
		fieldPath FieldPath
		wantErr   bool
	}{
		{
			desc:      "Invalid path",
			fieldPath: []string{},
			wantErr:   true,
		},
		{
			desc:      "Valid path",
			fieldPath: []string{"path"},
			wantErr:   false,
		},
	}
	for _, tc := range testcases {

		query := c.Collection("C")
		aggQuery := query.NewAggregationQuery().WithSumPath(tc.fieldPath, sumAlias)
		_, err := aggQuery.Get(ctx)
		if err == nil && tc.wantErr {
			t.Fatalf("%s: got nil wanted error", tc.desc)
		} else if err != nil && !tc.wantErr {
			t.Fatalf("%s: got %v, want nil", tc.desc, err)
		}
	}
}

func TestWithAvg(t *testing.T) {
	ctx := context.Background()
	avgAlias := "avg"
	c, srv, cleanup := newMock(t)
	defer cleanup()

	srv.addRPC(nil, []interface{}{
		&pb.RunAggregationQueryResponse{
			Result: &pb.AggregationResult{
				AggregateFields: map[string]*pb.Value{
					"avg": intval(1),
				},
			},
		},
	})

	testcases := []struct {
		desc    string
		path    string
		wantErr bool
	}{
		{
			desc:    "Invalid path",
			path:    "path*",
			wantErr: true,
		},
		{
			desc:    "Valid path",
			path:    "path",
			wantErr: false,
		},
	}
	for _, tc := range testcases {

		query := c.Collection("C")
		aggQuery := query.NewAggregationQuery().WithAvg(tc.path, avgAlias)
		_, err := aggQuery.Get(ctx)
		if err == nil && tc.wantErr {
			t.Fatalf("%s: got nil wanted error", tc.desc)
		} else if err != nil && !tc.wantErr {
			t.Fatalf("%s: got %v, want nil", tc.desc, err)
		}
	}
}

func TestWithAvgPath(t *testing.T) {
	ctx := context.Background()
	avgAlias := "avg"
	c, srv, cleanup := newMock(t)
	defer cleanup()

	srv.addRPC(nil, []interface{}{
		&pb.RunAggregationQueryResponse{
			Result: &pb.AggregationResult{
				AggregateFields: map[string]*pb.Value{
					"avg": intval(1),
				},
			},
		},
	})

	testcases := []struct {
		desc      string
		fieldPath FieldPath
		wantErr   bool
	}{
		{
			desc:      "Invalid path",
			fieldPath: []string{},
			wantErr:   true,
		},
		{
			desc:      "Valid path",
			fieldPath: []string{"path"},
			wantErr:   false,
		},
	}
	for _, tc := range testcases {

		query := c.Collection("C")
		aggQuery := query.NewAggregationQuery().WithAvgPath(tc.fieldPath, avgAlias)
		_, err := aggQuery.Get(ctx)
		if err == nil && tc.wantErr {
			t.Fatalf("%s: got nil wanted error", tc.desc)
		} else if err != nil && !tc.wantErr {
			t.Fatalf("%s: got %v, want nil", tc.desc, err)
		}
	}
}

func TestExplainOptionsApply(t *testing.T) {
	pbExplainOptions := pb.ExplainOptions{Analyze: true}
	for _, testcase := range []struct {
		desc            string
		existingOptions *pb.ExplainOptions
		newOptions      ExplainOptions
		wantErrMsg      string
	}{
		{
			desc:            "ExplainOptions specified multiple times",
			existingOptions: &pbExplainOptions,
			newOptions:      ExplainOptions{Analyze: true},
			wantErrMsg:      "ExplainOptions can be specified only once",
		},
		{
			desc:            "ExplainOptions specified once",
			existingOptions: nil,
			newOptions:      ExplainOptions{Analyze: true},
		},
	} {
		gotErr := testcase.newOptions.apply(&runQuerySettings{explainOptions: testcase.existingOptions})
		if (gotErr == nil && testcase.wantErrMsg != "") ||
			(gotErr != nil && !strings.Contains(gotErr.Error(), testcase.wantErrMsg)) {
			t.Errorf("%v: apply got: %v want: %v", testcase.desc, gotErr.Error(), testcase.wantErrMsg)
		}
	}
}

func TestNewRunQuerySettings(t *testing.T) {
	for _, testcase := range []struct {
		desc       string
		opts       []RunOption
		wantErrMsg string
	}{
		{
			desc:       "nil RunOption",
			opts:       []RunOption{ExplainOptions{Analyze: true}, nil},
			wantErrMsg: "cannot be nil",
		},
		{
			desc: "success RunOption",
			opts: []RunOption{ExplainOptions{Analyze: true}},
		},
		{
			desc:       "ExplainOptions specified multiple times",
			opts:       []RunOption{ExplainOptions{Analyze: true}, ExplainOptions{Analyze: false}, ExplainOptions{Analyze: true}},
			wantErrMsg: "ExplainOptions can be specified only once",
		},
	} {
		_, gotErr := newRunQuerySettings(testcase.opts)
		if (gotErr == nil && testcase.wantErrMsg != "") ||
			(gotErr != nil && !strings.Contains(gotErr.Error(), testcase.wantErrMsg)) {
			t.Errorf("%v: newRunQuerySettings got: %v want: %v", testcase.desc, gotErr, testcase.wantErrMsg)
		}
	}
}

func TestQueryRunOptionsAndGetAllWithOptions(t *testing.T) {
	ctx := context.Background()
	client, srv, cleanup := newMock(t)
	defer cleanup()

	dbPath := "projects/projectID/databases/(default)"
	collectionName := "collection01"
	wantReq := &pb.RunQueryRequest{
		Parent: fmt.Sprintf("%v/documents", dbPath),
		QueryType: &pb.RunQueryRequest_StructuredQuery{
			StructuredQuery: &pb.StructuredQuery{
				From: []*pb.StructuredQuery_CollectionSelector{
					{CollectionId: collectionName},
				},
			},
		},
		ExplainOptions: &pb.ExplainOptions{
			Analyze: false,
		},
	}

	srv.addRPC(wantReq, []interface{}{
		&pb.RunQueryResponse{
			Document: &pb.Document{
				Name:       fmt.Sprintf("%v/documents/%v/doc1", dbPath, collectionName),
				CreateTime: aTimestamp,
				UpdateTime: aTimestamp,
				Fields:     map[string]*pb.Value{"f": intval(2)},
			},
			ReadTime: aTimestamp,
		},
	})

	_, err := client.Collection(collectionName).
		WithRunOptions(ExplainOptions{Analyze: false}).
		WithRunOptions(ExplainOptions{Analyze: true}).
		WithRunOptions(ExplainOptions{Analyze: false}).
		Documents(ctx).
		GetAll()

	if err != nil {
		t.Fatal(err)
	}
}

func TestFindNearest(t *testing.T) {
	ctx := context.Background()
	c, srv, cleanup := newMock(t)
	defer cleanup()

	const dbPath = "projects/projectID/databases/(default)"
	mapFields := map[string]*pb.Value{
		typeKey: {ValueType: &pb.Value_StringValue{StringValue: typeValVector}},
		valueKey: {
			ValueType: &pb.Value_ArrayValue{
				ArrayValue: &pb.ArrayValue{
					Values: []*pb.Value{
						{ValueType: &pb.Value_DoubleValue{DoubleValue: 1}},
						{ValueType: &pb.Value_DoubleValue{DoubleValue: 2}},
						{ValueType: &pb.Value_DoubleValue{DoubleValue: 2}},
					},
				},
			},
		},
	}
	wantPBDocs := []*pb.Document{
		{
			Name:       dbPath + "/documents/C/a",
			CreateTime: aTimestamp,
			UpdateTime: aTimestamp,
			Fields:     map[string]*pb.Value{"EmbeddedField": mapval(mapFields)},
		},
	}

	testcases := []struct {
		desc        string
		path        string
		queryVector interface{}
		wantErr     bool
	}{
		{
			desc:    "Invalid path",
			path:    "path*",
			wantErr: true,
		},
		{
			desc:        "Valid path",
			path:        "path",
			queryVector: []float64{5, 6, 7},
			wantErr:     false,
		},
		{
			desc:        "Invalid vector type",
			path:        "path",
			queryVector: "abcd",
			wantErr:     true,
		},
		{
			desc:        "Valid vector type",
			path:        "path",
			queryVector: []float32{5, 6, 7},
			wantErr:     false,
		},
	}
	for _, tc := range testcases {
		srv.reset()
		srv.addRPC(nil, []interface{}{
			&pb.RunQueryResponse{Document: wantPBDocs[0]},
		})
		vQuery := c.Collection("C").FindNearest(tc.path, tc.queryVector, 2, DistanceMeasureEuclidean, nil)

		_, err := vQuery.Documents(ctx).GetAll()
		if err == nil && tc.wantErr {
			t.Fatalf("%s: got nil wanted error", tc.desc)
		} else if err != nil && !tc.wantErr {
			t.Fatalf("%s: got %v, want nil", tc.desc, err)
		}
	}
}
