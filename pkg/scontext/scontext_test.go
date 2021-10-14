// Stubs generated with github.com/hexdigest/gounit

package scontext

import (
	"context"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	ctx := context.Background()
	tests := []struct {
		name string
		args func(t *testing.T) args

		want1 StartStopContext
	}{
		{"New", func(t *testing.T) args { return args{ctx} }, StartStopContext{parentCtx: ctx}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			got1 := New(tArgs.ctx)

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("New got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

func TestStartStopContext_Context(t *testing.T) {
	ctx := context.Background()
	var ctxRet context.Context
	tests := []struct {
		name    string
		init    func(t *testing.T) *StartStopContext
		inspect func(r *StartStopContext, t *testing.T) //inspects receiver after test run

		want1 *context.Context
	}{
		{
			"New",
			func(t *testing.T) *StartStopContext {
				c := New(ctx)
				return &c
			},
			nil,
			&ctx,
		},
		{
			"Start",
			func(t *testing.T) *StartStopContext {
				c := New(ctx)
				var err error
				ctxRet, err = c.CreateContext()
				if err != nil {
					t.Errorf("Start failed with %v", err)
				}
				return &c
			},
			func(r *StartStopContext, t *testing.T) {

			},
			&ctxRet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.init(t)
			got1 := receiver.Context()

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

			if !reflect.DeepEqual(got1, *tt.want1) {
				t.Errorf("StartStopContext.Context got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

func TestStartStopContext_Start(t *testing.T) {
	var ctxRet, ctxRet1, nilCtx context.Context
	tests := []struct {
		name    string
		init    func(t *testing.T) *StartStopContext
		inspect func(r *StartStopContext, t *testing.T) //inspects receiver after test run

		want1      *context.Context
		wantErr    bool
		inspectErr func(err error, t *testing.T) //use for more precise error evaluation after test
	}{
		{
			"Normal",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				ctxRet = r.Context()
				if r.cancel == nil {
					t.Errorf("context cancel is nil")
				}
				if ctxRet == r.parentCtx {
					t.Errorf("internal context is parent context")
				}
			},
			&ctxRet,
			false,
			nil,
		},
		{
			"Restart",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				var err error
				ctxRet1, err = c.CreateContext()
				if err != nil {
					t.Errorf("context Start failed %v", err)
				}
				err = c.CancelContext()
				if err != nil {
					t.Errorf("context Stop failed %v", err)
				}
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				ctxRet = r.Context()
				if r.cancel == nil {
					t.Errorf("context cancel is nil")
				}
				if ctxRet == r.parentCtx {
					t.Errorf("internal context is parent context")
				}
				if ctxRet == ctxRet1 {
					t.Errorf("internal context is previous context")
				}
			},
			&ctxRet,
			false,
			nil,
		},
		{
			"Running",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				var err error
				ctxRet1, err = c.CreateContext()
				if err != nil {
					t.Errorf("context start failed: %v", err)
				}
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				ctx := r.Context()
				if r.cancel == nil {
					t.Errorf("context cancel is nil")
				}
				if ctx != ctxRet1 {
					t.Errorf("internal context was modified")
				}
			},
			&nilCtx,
			true,
			nil,
		},
		{
			"Cancelled",
			func(t *testing.T) *StartStopContext {
				ctx, cancel := context.WithCancel(context.Background())
				c := New(ctx)
				cancel()
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				if r.cancel != nil {
					t.Errorf("context cancel is not nil")
				}
			},
			&nilCtx,
			true,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.init(t)
			got1, err := receiver.CreateContext()

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

			if !reflect.DeepEqual(got1, *tt.want1) {
				t.Errorf("StartStopContext.Start got1 = %v, want1: %v", got1, tt.want1)
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("StartStopContext.Start error = %v, wantErr: %t", err, tt.wantErr)
			}

			if tt.inspectErr != nil {
				tt.inspectErr(err, t)
			}
		})
	}
}

func TestStartStopContext_Stop(t *testing.T) {
	tests := []struct {
		name    string
		init    func(t *testing.T) *StartStopContext
		inspect func(r *StartStopContext, t *testing.T) //inspects receiver after test run

		wantErr    bool
		inspectErr func(err error, t *testing.T) //use for more precise error evaluation after test
	}{
		{
			"Normal",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				return &c
			},
			nil,
			true,
			nil,
		},
		{
			"Running",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				if _, err := c.CreateContext(); err != nil {
					t.Errorf("context start failed: %v", err)
				}
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				if r.cancel != nil {
					t.Errorf("context cancel func not null")
				}
			},
			false,
			nil,
		},
		{
			"Stopped",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				if _, err := c.CreateContext(); err != nil {
					t.Errorf("context Start failed: %v", err)
				}
				if err := c.CancelContext(); err != nil {
					t.Errorf("context Stop failed: %v", err)
				}
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				if r.cancel != nil {
					t.Errorf("context cancel func not null")
				}
			},
			true,
			nil,
		},
		{
			"Restarted",
			func(t *testing.T) *StartStopContext {
				c := New(context.Background())
				if _, err := c.CreateContext(); err != nil {
					t.Errorf("context Start failed: %v", err)
				}
				if err := c.CancelContext(); err != nil {
					t.Errorf("context Stop failed: %v", err)
				}
				if _, err := c.CreateContext(); err != nil {
					t.Errorf("context Start failed: %v", err)
				}
				return &c
			},
			func(r *StartStopContext, t *testing.T) {
				if r.cancel != nil {
					t.Errorf("context cancel func null")
				}
			},
			false,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := tt.init(t)
			err := receiver.CancelContext()

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("StartStopContext.Stop error = %v, wantErr: %t", err, tt.wantErr)
			}

			if tt.inspectErr != nil {
				tt.inspectErr(err, t)
			}
		})
	}
}
