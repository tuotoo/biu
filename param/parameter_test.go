package param

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParameter_Bool(t *testing.T) {
	type fields struct {
		Value []string
		error error
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "parameter is nil",
			fields: struct {
				Value []string
				error error
			}{Value: nil},
			want:    false,
			wantErr: true,
		},
		{
			name: "parameter is 1",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"1"}},
			want:    true,
			wantErr: false,
		},
		{
			name: "parameter is 0",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"0"}},
			want:    false,
			wantErr: false,
		},
		{
			name: "parameter is 2",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"2"}},
			want:    false,
			wantErr: true,
		},
		{
			name: "parameter is A",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"A"}},
			want:    false,
			wantErr: true,
		},
		{
			name: "parameter is true",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"true"}},
			want:    true,
			wantErr: false,
		},
		{
			name: "parameter is True",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"True"}},
			want:    true,
			wantErr: false,
		},
		{
			name: "parameter is TRue",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"TRue"}},
			want:    false,
			wantErr: true,
		},
		{
			name: "parameter is false",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"false"}},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Parameter{
				value: tt.fields.Value,
				error: tt.fields.error,
			}
			got, err := p.Bool()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parameter.Bool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Parameter.Bool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameter_BoolDefault(t *testing.T) {
	type fields struct {
		Value []string
		error error
	}
	type args struct {
		defaultValue bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "parameter is nil",
			fields: struct {
				Value []string
				error error
			}{Value: nil},
			args: struct{ defaultValue bool }{defaultValue: true},
			want: true,
		},
		{
			name: "parameter is 1",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"1"}},
			args: struct{ defaultValue bool }{defaultValue: true},
			want: true,
		},
		{
			name: "parameter is 0",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"0"}},
			args: struct{ defaultValue bool }{defaultValue: true},
			want: false,
		},
		{
			name: "parameter is 2",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"2"}},
			args: struct{ defaultValue bool }{defaultValue: true},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Parameter{
				value: tt.fields.Value,
				error: tt.fields.error,
			}
			if got := p.BoolDefault(tt.args.defaultValue); got != tt.want {
				t.Errorf("Parameter.BoolDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameter_BoolArray(t *testing.T) {
	type fields struct {
		Value []string
		error error
	}
	tests := []struct {
		name    string
		fields  fields
		want    []bool
		wantErr bool
	}{
		{
			name: "parameter is nil",
			fields: struct {
				Value []string
				error error
			}{Value: nil},
			want:    []bool{},
			wantErr: false,
		},
		{
			name: "parameter is [true, 0]",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"true", "0"}},
			want:    []bool{true, false},
			wantErr: false,
		},
		{
			name: "parameter include invalid value",
			fields: struct {
				Value []string
				error error
			}{Value: []string{"true", "a"}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Parameter{
				value: tt.fields.Value,
				error: tt.fields.error,
			}
			got, err := p.BoolArray()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parameter.BoolArray() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parameter.BoolArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameter_Bytes(t *testing.T) {
	type fields struct {
		Value []string
		error error
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			fields: struct {
				Value []string
				error error
			}{Value: []string{"1"}},
			want:    []byte("1"),
			wantErr: false,
		},
		{
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Parameter{
				value: tt.fields.Value,
				error: tt.fields.error,
			}
			got, err := p.Bytes()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parameter.Bytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parameter.Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameter_BytesDefault(t *testing.T) {
	type fields struct {
		Value []string
		error error
	}
	type args struct {
		defaultValue []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		{
			args: struct{ defaultValue []byte }{defaultValue: []byte("1")},
			want: []byte("1"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Parameter{
				value: tt.fields.Value,
				error: tt.fields.error,
			}
			if got := p.BytesDefault(tt.args.defaultValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parameter.BytesDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParameter_Time(t *testing.T) {
	f := func(p Parameter, result time.Time, errIs error) {
		tt, err := p.Time(time.DateOnly)
		if errIs != nil {
			assert.ErrorAs(t, err, &errIs)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, result, tt)
	}
	f(Parameter{
		value: []string{"2024-01-06"},
	}, time.Date(2024, time.January, 6, 0, 0, 0, 0, time.UTC), nil)
	f(Parameter{
		value: nil,
	}, time.Time{}, ErrParamIsEmpty)
	f(Parameter{
		value: []string{"2024"},
	}, time.Time{}, &time.ParseError{})
	f(Parameter{
		error: ErrParamIsEmpty,
	}, time.Time{}, ErrParamIsEmpty)
}

func TestParameter_Int64(t *testing.T) {
	f := func(p Parameter, result int64, errContains string) {
		i, err := p.Int64()
		if errContains != "" {
			assert.ErrorContains(t, err, errContains)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, result, i)
	}
	f(Parameter{
		value: []string{"1"},
	}, 1, "")
	f(Parameter{
		value: nil,
	}, 0, "parameter is empty")
	f(Parameter{
		value: []string{"a"},
	}, 0, "invalid syntax")
	f(Parameter{
		error: ErrParamIsEmpty,
	}, 0, "parameter is empty")
}

func TestParameter_UInt64(t *testing.T) {
	f := func(p Parameter, result uint64, errContains string) {
		i, err := p.Uint64()
		if errContains != "" {
			assert.ErrorContains(t, err, errContains)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, result, i)
	}
	f(Parameter{
		value: []string{"1"},
	}, 1, "")
	f(Parameter{
		value: nil,
	}, 0, "parameter is empty")
	f(NewParameter([]string{"-1"}, nil), 1, "")
	f(Parameter{
		value: []string{"a"},
	}, 0, "invalid syntax")
	f(Parameter{
		error: ErrParamIsEmpty,
	}, 0, "parameter is empty")
}

func TestParameter_Int(t *testing.T) {
	f := func(p Parameter, result int, errContains string) {
		i, err := p.Int()
		if errContains != "" {
			assert.ErrorContains(t, err, errContains)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, result, i)
	}
	f(Parameter{
		value: []string{"1"},
	}, 1, "")
	f(Parameter{
		value: nil,
	}, 0, "parameter is empty")
	f(NewParameter([]string{"-1"}, nil), -1, "")
	f(Parameter{
		value: []string{"a"},
	}, 0, "invalid syntax")
	f(Parameter{
		error: ErrParamIsEmpty,
	}, 0, "parameter is empty")
}
