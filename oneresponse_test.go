package oneresponse

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	err2 = errors.New("some error")
	err3 = errors.New("another error")
)

var randomSeed = rand.New(rand.NewSource(time.Now().UnixNano()))

func successBoolFunc1(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		log.Error().Msg("context done")
		return false, ctx.Err()
	case <-time.After(time.Duration(randomSeed.Int31n(100)) * time.Millisecond):
		log.Info().Msg("returning true")
		break
	}
	return true, nil
}
func failingBoolFunc1(_ context.Context) (bool, error) {
	return false, err2
}
func failingBoolFunc2(_ context.Context) (bool, error) {
	return false, err3
}

func TestOneResponseSerial(t *testing.T) {
	type args[T any] struct {
		operation []OperationWithData[T]
	}
	type testCase[T any] struct {
		name        string
		args        args[T]
		want        T
		wantErr     bool
		expectedErr []*error
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
			want:        false,
			wantErr:     true,
			expectedErr: []*error{&err2, &err3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := Serial(ctx, tt.args.operation)
			if tt.wantErr {
				require.Error(t, err, fmt.Sprintf("Serial(%v)", tt.args.operation))
			}
			require.Equalf(t, tt.want, got, "Serial(%v)", tt.args.operation)
			if tt.wantErr {
				for _, e := range tt.expectedErr {
					require.ErrorAs(t, err, e)
				}
			}
		})
	}
}

func TestOneResponseParallel(t *testing.T) {
	type args[T any] struct {
		operation []OperationWithData[T]
	}
	type testCase[T any] struct {
		name        string
		args        args[T]
		want        T
		wantErr     bool
		expectedErr []*error
	}
	tests := []testCase[bool]{
		{
			name: "bool",
			args: args[bool]{
				operation: []OperationWithData[bool]{
					failingBoolFunc1,
					successBoolFunc1,
					successBoolFunc1,
					successBoolFunc1,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "bool with error",
			args: args[bool]{
				operation: []OperationWithData[bool]{failingBoolFunc1, failingBoolFunc2},
			},
			want:        false,
			wantErr:     true,
			expectedErr: []*error{&err2, &err3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got, err := Parallel(ctx, tt.args.operation)
			if tt.wantErr {
				require.Error(t, err, fmt.Sprintf("Parallel(%v)", tt.args.operation))
			}
			require.Equalf(t, tt.want, got, "Parallel(%v)", tt.args.operation)
			if tt.wantErr {
				for _, e := range tt.expectedErr {
					require.ErrorAs(t, err, e)
				}
			}
		})
	}
}
