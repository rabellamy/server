package config

import (
	"reflect"

	"github.com/kelseyhightower/envconfig"
)

// LoadConfig loads configuration from environment variables into a struct of type T.
// It uses the provided prefix to scope environment variables (e.g. PREFIX_VAR).
// If the struct T has a field named "Namespace" of type string and it is empty
// after loading, it will be set to the value of the prefix.
func LoadConfig[T any](prefix string) (T, error) {
	var c T
	// Load environment variables
	if err := envconfig.Process(prefix, &c); err != nil {
		return c, err
	}

	// Use reflection to set the default Namespace if it's empty
	v := reflect.ValueOf(&c).Elem()
	if v.Kind() == reflect.Struct {
		ns := v.FieldByName("Namespace")
		if ns.IsValid() && ns.Kind() == reflect.String && ns.CanSet() {
			if ns.String() == "" {
				ns.SetString(prefix)
			}
		}
	}

	return c, nil
}
