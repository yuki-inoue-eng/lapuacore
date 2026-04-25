package mutex

import (
	"reflect"
	"testing"
)

func TestSlice_AppendHead(t *testing.T) {
	type input struct {
		init   []int
		maxLen int
		val    int
	}
	type testCase struct {
		name     string
		input    input
		expected *Slice[int]
	}
	tests := []testCase{
		{
			name: "case 1",
			input: input{
				init:   nil,
				maxLen: -1,
				val:    9,
			},
			expected: NewSlice[int]([]int{9}, -1),
		},
		{
			name: "case 2",
			input: input{
				init:   []int{1, 2, 3},
				maxLen: 3,
				val:    9,
			},
			expected: NewSlice[int]([]int{9, 1, 2}, 3),
		},
	}
	for _, tt := range tests {
		s := NewSlice[int](tt.input.init, tt.input.maxLen)
		t.Run(tt.name, func(t *testing.T) {
			s.AppendHead(tt.input.val)
			if !reflect.DeepEqual(s.a, tt.expected.a) {
				t.Errorf("got %v, want %v", s.a, tt.expected.a)
			}
		})
	}
}

func TestSlice_Append(t *testing.T) {
	type input struct {
		init   []int
		maxLen int
		val    int
	}
	type testCase struct {
		name     string
		input    input
		expected *Slice[int]
	}
	tests := []testCase{
		{
			name: "case 1",
			input: input{
				init:   nil,
				maxLen: -1,
				val:    9,
			},
			expected: NewSlice[int]([]int{9}, -1),
		},
		{
			name: "case 2",
			input: input{
				init:   []int{1, 2, 3},
				maxLen: 3,
				val:    9,
			},
			expected: NewSlice[int]([]int{2, 3, 9}, 3),
		},
	}
	for _, tt := range tests {
		s := NewSlice[int](tt.input.init, tt.input.maxLen)
		t.Run(tt.name, func(t *testing.T) {
			s.Append(tt.input.val)
			if !reflect.DeepEqual(s.a, tt.expected.a) {
				t.Errorf("got %v, want %v", s.a, tt.expected.a)
			}
		})
	}
}

func Test_Get(t *testing.T) {
	type testCase struct {
		name     string
		inputIdx int
		expected int
	}
	tests := []testCase{
		{
			name:     "normal case 1",
			inputIdx: 0,
			expected: 10,
		},
		{
			name:     "normal case 2",
			inputIdx: 1,
			expected: 20,
		},
		{
			name:     "normal case 3",
			inputIdx: -1,
			expected: 50,
		},
		{
			name:     "normal case 4",
			inputIdx: -2,
			expected: 40,
		},
	}
	for _, tt := range tests {
		testTarget := NewSlice[int]([]int{10, 20, 30, 40, 50}, -1)
		t.Run(tt.name, func(t *testing.T) {
			actualVal := testTarget.Get(tt.inputIdx)
			if got, want := actualVal, tt.expected; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}
