package conf

import (
	"fmt"
	"reflect"

	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/builtin"
	"github.com/antonmedv/expr/vm/runtime"
)

type FunctionTable map[string]*builtin.Function

type Config struct {
	Env         interface{}
	Types       TypesTable
	MapEnv      bool
	DefaultType reflect.Type
	Operators   OperatorsTable
	Expect      reflect.Kind
	ExpectAny   bool
	Optimize    bool
	Strict      bool
	ConstFns    map[string]reflect.Value
	Visitors    []ast.Visitor
	Functions   FunctionTable
	Pipes       bool
}

// CreateNew creates new config with default values.
func CreateNew() *Config {
	c := &Config{
		Operators: make(OperatorsTable),
		ConstFns:  make(map[string]reflect.Value),
		Functions: make(FunctionTable),
		Optimize:  true,
	}
	for _, f := range builtin.Functions {
		c.Functions[f.Name] = f
	}
	return c
}

// New creates new config with environment.
func New(env interface{}) *Config {
	c := CreateNew()
	c.WithEnv(env)
	return c
}

func (c *Config) WithEnv(env interface{}) {
	var mapEnv bool
	var mapValueType reflect.Type
	if _, ok := env.(map[string]interface{}); ok {
		mapEnv = true
	} else {
		if reflect.ValueOf(env).Kind() == reflect.Map {
			mapValueType = reflect.TypeOf(env).Elem()
		}
	}

	c.Env = env
	c.Types = CreateTypesTable(env)
	c.MapEnv = mapEnv
	c.DefaultType = mapValueType
	c.Strict = true
}

func (c *Config) Operator(operator string, fns ...string) {
	c.Operators[operator] = append(c.Operators[operator], fns...)
}

func (c *Config) ConstExpr(name string) {
	if c.Env == nil {
		panic("no environment is specified for ConstExpr()")
	}
	fn := reflect.ValueOf(runtime.Fetch(c.Env, name))
	if fn.Kind() != reflect.Func {
		panic(fmt.Errorf("const expression %q must be a function", name))
	}
	c.ConstFns[name] = fn
}

func (c *Config) Check() {
	for operator, fns := range c.Operators {
		for _, fn := range fns {
			fnType, foundType := c.Types[fn]
			fnFunc, foundFunc := c.Functions[fn]
			if !foundFunc && (!foundType || fnType.Type.Kind() != reflect.Func) {
				panic(fmt.Errorf("function %s for %s operator does not exist in the environment", fn, operator))
			}

			if foundType {
				checkType(fnType, fn, operator)
			}
			if foundFunc {
				checkFunc(fnFunc, fn, operator)
			}
		}
	}
}

func checkType(fnType Tag, fn string, operator string) {
	requiredNumIn := 2
	if fnType.Method {
		requiredNumIn = 3 // As first argument of method is receiver.
	}
	if fnType.Type.NumIn() != requiredNumIn || fnType.Type.NumOut() != 1 {
		panic(fmt.Errorf("function %s for %s operator does not have a correct signature", fn, operator))
	}
}

func checkFunc(fn *builtin.Function, name string, operator string) {
	if len(fn.Types) == 0 {
		panic(fmt.Errorf("function %s for %s operator misses types", name, operator))
	}
	for _, t := range fn.Types {
		if t.NumIn() != 2 || t.NumOut() != 1 {
			panic(fmt.Errorf("function %s for %s operator does not have a correct signature", name, operator))
		}
	}
}
