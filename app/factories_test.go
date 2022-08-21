package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoDepDeclared(t *testing.T) {
	_, err := dependencies{}.resolve("missing",
		instances{}.With("any", 1))
	assert.EqualError(t, err, "missing is not declared")
}

func TestNotAFunctionInFactories(t *testing.T) {
	_, err := Factories{"foo": 1}.dependencies()
	assert.EqualError(t, err, "foo is not a function")
}

func TestNotAFactoryFunction(t *testing.T) {
	_, err := Factories{"foo": func() (int, int, int) {
		return 1, 2, 3
	}}.dependencies()
	assert.EqualError(t, err, "foo is not a factory")
}

func TestTransitiveDependencyFails(t *testing.T) {
	deps, err := Factories{
		"a": func(b *serviceA) *mainServer {
			return nil
		},
		"b": func() (*serviceA, error) {
			return nil, fmt.Errorf("nope")
		},
	}.dependencies()
	assert.NoError(t, err)
	_, err = deps.resolve("a", instances{})
	assert.EqualError(t, err, "cannot resolve a because of b: nope")
}

func TestCannotFindDependency(t *testing.T) {
	deps, err := Factories{
		"a": func(b *serviceA) *mainServer {
			return nil
		},
	}.dependencies()
	assert.NoError(t, err)
	_, err = deps.resolve("a", instances{})
	assert.EqualError(t, err, "cannot find *app.serviceA for a")
}
