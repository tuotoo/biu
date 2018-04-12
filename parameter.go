package biu

import (
	"math/big"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type Parameter struct {
	Value []string
	error
}

func (p Parameter) Bool() (bool, error) {
	var zeroVal bool
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		return strconv.ParseBool(p.Value[0])
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) BoolDefault(defaultValue bool) bool {
	rst, err := p.Bool()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) BoolArray() ([]bool, error) {
	var rst []bool
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]bool, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseBool(v)
		if err != nil {
			return rst, err
		}
		rst[i] = m
	}
	return rst, nil
}

func (p Parameter) Float32() (float32, error) {
	var zeroVal float32
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseFloat(p.Value[0], 32)
		return float32(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Float32Default(defaultValue float32) float32 {
	rst, err := p.Float32()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Float32Array() ([]float32, error) {
	var rst []float32
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]float32, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return rst, err
		}
		rst[i] = float32(m)
	}
	return rst, nil
}

func (p Parameter) Float64() (float64, error) {
	var zeroVal float64
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		return strconv.ParseFloat(p.Value[0], 64)
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Float64Default(defaultValue float64) float64 {
	rst, err := p.Float64()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Float64Array() ([]float64, error) {
	var rst []float64
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]float64, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return rst, err
		}
		rst[i] = m
	}
	return rst, nil
}

func (p Parameter) Int() (int, error) {
	var zeroVal int
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseInt(p.Value[0], 10, 32)
		return int(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) IntDefault(defaultValue int) int {
	rst, err := p.Int()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) IntArray() ([]int, error) {
	var rst []int
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]int, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return rst, err
		}
		rst[i] = int(m)
	}
	return rst, nil
}

func (p Parameter) Int8() (int8, error) {
	var zeroVal int8
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseInt(p.Value[0], 10, 8)
		return int8(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Int8Default(defaultValue int8) int8 {
	rst, err := p.Int8()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Int8Array() ([]int8, error) {
	var rst []int8
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]int8, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return rst, err
		}
		rst[i] = int8(m)
	}
	return rst, nil
}

func (p Parameter) Int16() (int16, error) {
	var zeroVal int16
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseInt(p.Value[0], 10, 16)
		return int16(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Int16Default(defaultValue int16) int16 {
	rst, err := p.Int16()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Int16Array() ([]int16, error) {
	var rst []int16
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]int16, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return rst, err
		}
		rst[i] = int16(m)
	}
	return rst, nil
}

func (p Parameter) Int32() (int32, error) {
	var zeroVal int32
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseInt(p.Value[0], 10, 32)
		return int32(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Int32Default(defaultValue int32) int32 {
	rst, err := p.Int32()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Int32Array() ([]int32, error) {
	var rst []int32
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]int32, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return rst, err
		}
		rst[i] = int32(m)
	}
	return rst, nil
}

func strToInt64(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		i := new(big.Int)
		ni, ok := i.SetString(s, 10) // octal
		if !ok {
			return v, err
		}
		return ni.Int64(), nil
	}
	return v, err
}

func (p Parameter) Int64() (int64, error) {
	var zeroVal int64
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		return strToInt64(p.Value[0])
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Int64Default(defaultValue int64) int64 {
	rst, err := p.Int64()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Int64Array() ([]int64, error) {
	var rst []int64
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]int64, len(p.Value))
	for i, v := range p.Value {
		m, err := strToInt64(v)
		if err != nil {
			return rst, err
		}
		rst[i] = m
	}
	return rst, nil
}

func (p Parameter) Uint() (uint, error) {
	var zeroVal uint
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseUint(p.Value[0], 10, 32)
		return uint(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) UintDefault(defaultValue uint) uint {
	rst, err := p.Uint()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) UintArray() ([]uint, error) {
	var rst []uint
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]uint, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return rst, err
		}
		rst[i] = uint(m)
	}
	return rst, nil
}

func (p Parameter) Uint8() (uint8, error) {
	var zeroVal uint8
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseUint(p.Value[0], 10, 8)
		return uint8(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Uint8Default(defaultValue uint8) uint8 {
	rst, err := p.Uint8()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Uint8Array() ([]uint8, error) {
	var rst []uint8
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]uint8, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return rst, err
		}
		rst[i] = uint8(m)
	}
	return rst, nil
}

func (p Parameter) Uint16() (uint16, error) {
	var zeroVal uint16
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseUint(p.Value[0], 10, 16)
		return uint16(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Uint16Default(defaultValue uint16) uint16 {
	rst, err := p.Uint16()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Uint16Array() ([]uint16, error) {
	var rst []uint16
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]uint16, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return rst, err
		}
		rst[i] = uint16(m)
	}
	return rst, nil
}

func (p Parameter) Uint32() (uint32, error) {
	var zeroVal uint32
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		v, err := strconv.ParseUint(p.Value[0], 10, 32)
		return uint32(v), err
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Uint32Default(defaultValue uint32) uint32 {
	rst, err := p.Uint32()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Uint32Array() ([]uint32, error) {
	var rst []uint32
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]uint32, len(p.Value))
	for i, v := range p.Value {
		m, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return rst, err
		}
		rst[i] = uint32(m)
	}
	return rst, nil
}

func strToUint64(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		i := new(big.Int)
		ni, ok := i.SetString(s, 10) // octal
		if !ok {
			return v, err
		}
		return ni.Uint64(), nil
	}
	return v, err
}

func (p Parameter) Uint64() (uint64, error) {
	var zeroVal uint64
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		return strToUint64(p.Value[0])
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) Uint64Default(defaultValue uint64) uint64 {
	rst, err := p.Uint64()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) Uint64Array() ([]uint64, error) {
	var rst []uint64
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]uint64, len(p.Value))
	for i, v := range p.Value {
		m, err := strToUint64(v)
		if err != nil {
			return rst, err
		}
		rst[i] = m
	}
	return rst, nil
}

func (p Parameter) String() (string, error) {
	var zeroVal string
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		return p.Value[0], nil
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) StringDefault(defaultValue string) string {
	rst, err := p.String()
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) StringArray() ([]string, error) {
	var rst []string
	if p.error != nil {
		return rst, p.error
	}
	return p.Value, nil
}

func (p Parameter) Time(layout string) (time.Time, error) {
	var zeroVal time.Time
	if p.error != nil {
		return zeroVal, p.error
	}
	if len(p.Value) > 0 {
		return time.Parse(layout, p.Value[0])
	}
	return zeroVal, errors.New("parameter is empty")
}

func (p Parameter) TimeDefault(layout string, defaultValue time.Time) time.Time {
	rst, err := p.Time(layout)
	if err != nil {
		return defaultValue
	}
	return rst
}

func (p Parameter) TimeArray(layout string) ([]time.Time, error) {
	var rst []time.Time
	if p.error != nil {
		return rst, p.error
	}
	rst = make([]time.Time, len(p.Value))
	for i, v := range p.Value {
		m, err := time.Parse(layout, v)
		if err != nil {
			return rst, err
		}
		rst[i] = m
	}
	return rst, nil
}
