package oneresponse

import (
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test functions and errors
var (
	err2             = errors.New("some error")
	err3             = errors.New("another error")
	successBoolFunc1 = func() (bool, error) {
		return true, nil
	}
	failingBoolFunc1 = func() (bool, error) {
		return false, err2
	}
	failingBoolFunc2 = func() (bool, error) {
		return false, err3
	}
)

func TestOneResponseSerial(t *testing.T) {
	type args[T any] struct {
		operation []OperationWithData[T]
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		want    T
		wantErr bool
	}
	tests := []testCase[bool]{
		{
			name: "bool",
			args: args[bool]{
				operation: []OperationWithData[bool]{failingBoolFunc1, successBoolFunc1},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "bool with error",
			args: args[bool]{
				operation: []OperationWithData[bool]{failingBoolFunc1, failingBoolFunc2},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Serial(tt.args.operation)
			if tt.wantErr {
				require.Error(t, err, fmt.Sprintf("Serial(%v)", tt.args.operation))
			}
			require.Equalf(t, tt.want, got, "Serial(%v)", tt.args.operation)
			if tt.wantErr {
				require.Equal(t, errors.Join(err2, err3), err)
			}
		})
	}
}

func TestOneResponseParallel(t *testing.T) {
	type args[T any] struct {
		operation []OperationWithData[T]
	}
	type testCase[T any] struct {
		name    string
		args    args[T]
		want    T
		wantErr bool
	}
	tests := []testCase[bool]{
		{
			name: "bool",
			args: args[bool]{
				operation: []OperationWithData[bool]{failingBoolFunc1, successBoolFunc1},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "bool with error",
			args: args[bool]{
				operation: []OperationWithData[bool]{failingBoolFunc1, failingBoolFunc2},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parallel(tt.args.operation)
			if tt.wantErr {
				require.Error(t, err, fmt.Sprintf("Parallel(%v)", tt.args.operation))
			}
			require.Equalf(t, tt.want, got, "Parallel(%v)", tt.args.operation)
			if tt.wantErr {
				expectedErr := []*error{&err2, &err3}
				for _, e := range expectedErr {
					require.ErrorAs(t, err, e)
				}
			}
			time.Sleep(100 * time.Millisecond)
		})
	}
}
