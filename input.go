package main

import (
	"fmt"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type InputTypes interface {
	int | float64
}

type InputValue[T InputTypes] struct {
	Value T
	Text  string
	Edit  bool
}

func NewInputValue[T InputTypes](value T) *InputValue[T] {
	i := &InputValue[T]{Value: value}
	i.UpdateText()
	return i
}

func (i *InputValue[_]) ToggleEdit() {
	i.Edit = !i.Edit
}

func (i *InputValue[T]) UpdateText() {
	i.Text = i.String()
}

func (i *InputValue[T]) String() string {
	switch v := any(i.Value).(type) {
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', 2, 64)
	default:
		return fmt.Sprint(i.Value)
	}
}

func InputsToVec3[T InputTypes](in [3]*InputValue[T]) rl.Vector3 {
	return rl.NewVector3(float32(in[0].Value), float32(in[1].Value), float32(in[2].Value))
}

func InputsToSlice3[T InputTypes](in [3]*InputValue[T]) [3]T {
	return [3]T{in[0].Value, in[1].Value, in[2].Value}
}
