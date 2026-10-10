// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// fieldSpec describes one Config field discovered by walkConfig(), resolved from its
// struct tags. It is the single point where a field's config-file key, CLI flag, and
// environment variable are read, so the three consumers cannot drift apart.
// issue: https://github.com/open-telemetry/opentelemetry-operator/issues/3565
type fieldSpec struct {
	// the field; addressable when walking a *Config
	value reflect.Value
	// full config-file key, dotted for nested structs; "" when unset
	yaml string
	// CLI flag name; "" or "-" means none
	flag string
	// environment variable name; "" or "-" means none
	env string
	// flag help and documentation text
	usage string
	// documentation section; "" excludes the field from the reference doc
	group string
	// []string registered as pflag StringArray (no comma split) rather than StringSlice
	array bool
	// flag registered outside the generic walker
	manual bool
}

// Nested config-types are the struct-valued Config fields the walker recurses into. Any
// other struct field (Internal, the Instrumentation CRD type, []corev1.EnvVar) is
// treated as an opaque leaf and skipped by the flag/env consumers.
func isNestedConfig(t reflect.Type) bool {
	switch t {
	case reflect.TypeFor[TLSConfig](), reflect.TypeFor[ZapConfig]():
		return true
	}
	return false
}

// walkConfig visits every leaf field of a Config struct value, recursing into the
// nested config structs and carrying the dotted yaml prefix so nested keys (e.g.
// tls.minversion) can be reconstructed for documentation.
func walkConfig(v reflect.Value, prefix string, visit func(fieldSpec)) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := v.Field(i)
		key := yamlName(ft)
		if isNestedConfig(ft.Type) {
			np := prefix
			if key != "" {
				np = key + "."
			}
			walkConfig(fv, np, visit)
			continue
		}
		fullYAML := ""
		if key != "" {
			fullYAML = prefix + key
		}
		flagType := ft.Tag.Get("flagtype")
		visit(fieldSpec{
			value:  fv,
			yaml:   fullYAML,
			flag:   active(ft.Tag.Get("flag")),
			env:    active(ft.Tag.Get("env")),
			usage:  ft.Tag.Get("usage"),
			group:  ft.Tag.Get("group"),
			array:  flagType == "array",
			manual: flagType == "manual",
		})
	}
}

// yamlName returns the config-file key for a field: the yaml tag when present, or the
// lowercased field name otherwise. It returns ""
// for fields excluded from the config file (yaml:"-").
func yamlName(ft reflect.StructField) string {
	tag, ok := ft.Tag.Lookup("yaml")
	if !ok {
		return strings.ToLower(ft.Name)
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	return name
}

// active normalizes a flag/env tag value: an empty tag or "-" means the mechanism is
// not used for the field.
func active(tag string) string {
	if tag == "-" {
		return ""
	}
	return tag
}

// registerFlags registers a pflag for every field carrying a flag tag, reading the
// default value from the provided (defaulted) Config.
// NOTE: Fields marked manual are left for their caller to register.
func registerFlags(f *pflag.FlagSet, defaults Config) {
	walkConfig(reflect.ValueOf(defaults), "", func(fs fieldSpec) {
		if fs.flag == "" || fs.manual {
			return
		}
		switch fs.value.Kind() {
		case reflect.String:
			f.String(fs.flag, fs.value.String(), fs.usage)
		case reflect.Bool:
			f.Bool(fs.flag, fs.value.Bool(), fs.usage)
		case reflect.Int:
			f.Int(fs.flag, int(fs.value.Int()), fs.usage)
		case reflect.Int32:
			v, _ := reflect.TypeAssert[int32](fs.value)
			f.Int32(fs.flag, v, fs.usage)
		case reflect.Slice:
			if def, ok := reflect.TypeAssert[[]string](fs.value); ok {
				// Copy the default so the registered flag never aliases the New() slice.
				def = append([]string(nil), def...)
				if fs.array {
					f.StringArray(fs.flag, def, fs.usage)
				} else {
					f.StringSlice(fs.flag, def, fs.usage)
				}
			}
		default:
		}
	})
}

// applyFlags copies every changed flag value back into the matching Config field.
func applyFlags(cfg *Config, f *pflag.FlagSet) {
	byFlag := map[string]reflect.Value{}
	walkConfig(reflect.ValueOf(cfg).Elem(), "", func(fs fieldSpec) {
		if fs.flag != "" && !fs.manual {
			byFlag[fs.flag] = fs.value
		}
	})
	f.Visit(func(fl *pflag.Flag) {
		fv, ok := byFlag[fl.Name]
		if !ok {
			return
		}
		switch fl.Value.Type() {
		case "string":
			fv.SetString(fl.Value.String())
		case "bool":
			b, _ := strconv.ParseBool(fl.Value.String())
			fv.SetBool(b)
		case "int", "int32":
			n, _ := strconv.ParseInt(fl.Value.String(), 10, 64)
			fv.SetInt(n)
		case "stringArray", "stringSlice":
			if sv, ok := fl.Value.(pflag.SliceValue); ok {
				fv.Set(reflect.ValueOf(sv.GetSlice()))
			}
		}
	})
}

// applyEnv overlays environment variables onto the Config for every field carrying an
// env tag.
func applyEnv(cfg *Config) {
	walkConfig(reflect.ValueOf(cfg).Elem(), "", func(fs fieldSpec) {
		if fs.env == "" {
			return
		}
		v, ok := os.LookupEnv(fs.env)
		if !ok {
			return
		}
		switch fs.value.Kind() {
		case reflect.String:
			fs.value.SetString(v)
		case reflect.Bool:
			b, _ := strconv.ParseBool(v)
			fs.value.SetBool(b)
		case reflect.Int:
			n, _ := strconv.Atoi(v)
			fs.value.SetInt(int64(n))
		case reflect.Int32:
			if n, err := strconv.ParseInt(v, 10, 32); err == nil {
				fs.value.SetInt(n)
			}
		case reflect.Slice:
			if _, ok := reflect.TypeAssert[[]string](fs.value); ok {
				fs.value.Set(reflect.ValueOf(strings.Split(v, ",")))
			}
		default:
		}
	})
}

// DocField is a documentation view of one configurable field, used by the config-docs
// generator. It is produced from the same tags that drive the runtime consumers.
// issue: https://github.com/open-telemetry/opentelemetry-operator/issues/3565
type DocField struct {
	Group   string
	YAML    string
	Flag    string
	Env     string
	Default string
	Usage   string
}

// DocFields returns the documentable fields.
func DocFields() []DocField {
	var out []DocField
	walkConfig(reflect.ValueOf(New()), "", func(fs fieldSpec) {
		if fs.group == "" {
			return
		}
		def := defaultString(fs.value)
		if fs.manual {
			def = ""
		}
		out = append(out, DocField{
			Group:   fs.group,
			YAML:    fs.yaml,
			Flag:    fs.flag,
			Env:     fs.env,
			Default: def,
			Usage:   fs.usage,
		})
	})
	return out
}

func defaultString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Slice:
		if s, ok := reflect.TypeAssert[[]string](v); ok {
			return strings.Join(s, ",")
		}
	default:
	}
	return ""
}
