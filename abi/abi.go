package abi

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/0x10f/go-tron/address"
	"io/ioutil"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

type ABI struct {
	Constructor Function
	Functions   map[string]Function
	Events      map[string]Event
}

func ReadFile(path string) (ABI, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return ABI{}, nil
	}

	var abi ABI
	if err := json.Unmarshal(file, &abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

func (a *ABI) UnmarshalJSON(data []byte) error {
	a.Functions = make(map[string]Function)
	a.Events = make(map[string]Event)

	type entry struct {
		Type       string  `json:"type"`
		Name       string  `json:"name"`
		Mutability string  `json:"stateMutability"`
		Inputs     []Value `json:"inputs"`
		Outputs    []Value `json:"outputs"`
	}

	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		switch entry.Type {
		case "Constructor":
			a.Constructor = Function{
				Name:       entry.Name,
				Mutability: entry.Mutability,
				Inputs:     entry.Inputs,
				Outputs:    entry.Outputs,
			}
		case "Function":
			a.Functions[entry.Name] = Function{
				Name:       entry.Name,
				Mutability: entry.Mutability,
				Inputs:     entry.Inputs,
				Outputs:    entry.Outputs,
			}
		case "Event":
			a.Events[entry.Name] = Event{
				Name:   entry.Name,
				Inputs: entry.Inputs,
			}
		}
	}

	return nil
}

type Function struct {
	Name       string
	Mutability string
	Inputs     []Value
	Outputs    []Value
}

func (f Function) Signature() string {
	var str strings.Builder
	str.WriteString(f.Name)

	str.WriteRune('(')
	for i, in := range f.Inputs {
		if i > 0 {
			str.WriteRune(',')
		}
		str.Write([]byte(in.Type))
	}
	str.WriteRune(')')

	return str.String()
}

// Payable returns if the function accepts Tron.
func (f Function) Payable() bool {
	return f.Mutability == "payable"
}

// Immutable returns if the function updates the blockchain on execution.
func (f Function) Immutable() bool {
	switch f.Mutability {
	case "pure", "view":
		return true
	default:
		return false
	}
}

const alignment = 32

func (f Function) Encode(args ...interface{}) []byte {
	var buf bytes.Buffer
	for _, arg := range args {
		switch arg := arg.(type) {
		case uint8, uint16, uint32, uint64:
			leftPad(&buf, 0x00, alignment-binary.Size(arg))
			binary.Write(&buf, binary.BigEndian, arg)
		case address.Address:
			leftPad(&buf, 0x00, alignment-len(arg)+1)
			buf.Write(arg[1:])
		case *big.Int:
			b := arg.Bytes()
			switch arg.Sign() {
			case -1:
				leftPad(&buf, 0xff, alignment-len(b))
			default:
				leftPad(&buf, 0x00, alignment-len(b))
			}
			buf.Write(arg.Bytes())
		default:
			panic("abi: cannot encode given argument, unsupported type")
		}
	}
	return buf.Bytes()
}

func leftPad(buf *bytes.Buffer, b byte, n int) {
	var fill [alignment]byte
	for i := range fill {
		fill[i] = b
	}
	buf.Write(fill[:n])
}

func (f Function) Decode(b []byte) ([]interface{}, error) {
	result := make([]interface{}, 0, len(f.Outputs))

	r := bytes.NewReader(b)

	var bs [alignment]byte
	for _, out := range f.Outputs {
		switch out.Type {
		case TypeBool:
			if _, err := r.Read(bs[:]); err != nil {
				return nil, err
			}

			switch b[alignment-1] {
			case 0:
				result = append(result, false)
			default:
				result = append(result, true)
			}
		case TypeBytes32:
			var slice [32]byte
			if _, err := r.Read(slice[:]); err != nil {
				return nil, err
			}
			result = append(result, slice)
		case TypeUint256:
			if _, err := r.Read(bs[:]); err != nil {
				return nil, err
			}
			result = append(result, big.NewInt(0).SetBytes(b[:]))
		}
	}

	return result, nil
}

func (f Function) GetOutputIndex(name string) int {
	for i, out := range f.Outputs {
		if out.Name == name {
			return i
		}
	}
	return -1
}

type Event struct {
	Name   string
	Inputs []Value
}

type Value struct {
	Name    string    `json:"name"`
	Type    ValueType `json:"type"`
	Indexed bool      `json:"indexed"`
}

type ValueType string

const (
	TypeBool    ValueType = "bool"
	TypeBytes32 ValueType = "bytes32"
	TypeUint256 ValueType = "uint256"
)

func Unmarshal(data []byte, fn Function, v interface{}) error {
	values, err := fn.Decode(data)
	if err != nil {
		return err
	}

	reflected := reflect.ValueOf(v).Elem()
	t := reflect.TypeOf(v).Elem()

	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag

		selector := tag.Get("abi")

		var index int
		switch {
		case strings.HasPrefix(selector, "$"):
			index, err = strconv.Atoi(selector[1:])
			if err != nil {
				return err
			}
		default:
			index = fn.GetOutputIndex(selector)
		}

		if index == -1 {
			continue
		}

		// TODO(271): Assure value is assignable to structure field.
		reflected.Field(i).Set(reflect.ValueOf(values[index]))
	}

	return nil
}