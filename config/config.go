package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Load reads .env if present, then environment variables, into out (pointer to struct).
func Load(out any) error {
	_ = godotenv.Load() // best-effort: only populates process env if .env exists

	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return errors.New("configx: out must be pointer to struct")
	}
	v = v.Elem()
	t := v.Type()

	var errs []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		// Nested struct support (one level)
		if f.Type.Kind() == reflect.Struct && f.Anonymous == false {
			subPtr := v.Field(i).Addr().Interface()
			if err := Load(subPtr); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", f.Name, err))
			}
			continue
		}

		envName := f.Tag.Get("env")
		if envName == "" {
			// default to field name in upper snake: AppPort -> APP_PORT
			envName = toEnvName(f.Name)
		}
		raw, ok := os.LookupEnv(envName)
		if !ok {
			if def := f.Tag.Get("default"); def != "" {
				raw = def
				ok = true
			}
		}
		req := f.Tag.Get("required") == "true"

		if !ok {
			if req {
				errs = append(errs, fmt.Sprintf("missing required %q", envName))
			}
			continue
		}

		if err := setField(v.Field(i), raw, f.Tag.Get("split")); err != nil {
			errs = append(errs, fmt.Sprintf("%s (%s): %v", f.Name, envName, err))
		}
	}

	if len(errs) > 0 {
		return errors.New("configx: \n - " + strings.Join(errs, "\n - "))
	}
	return nil
}

// MustLoad is a convenience wrapper that panics on error.
func MustLoad(out any) {
	if err := Load(out); err != nil {
		panic(err)
	}
}

func setField(fv reflect.Value, raw string, split string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid bool %q", raw)
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special-case time.Duration
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("invalid duration %q", raw)
			}
			fv.SetInt(int64(d))
			return nil
		}
		i, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid int %q", raw)
		}
		fv.SetInt(i)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element type %s", fv.Type().Elem().Kind())
		}
		sep := ","
		if split != "" {
			sep = split
		}
		parts := strings.Split(raw, sep)
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		fv.Set(reflect.ValueOf(out))
	default:
		return fmt.Errorf("unsupported kind %s", fv.Kind())
	}
	return nil
}

func toEnvName(field string) string {
	var b strings.Builder
	for i, r := range field {
		if i > 0 && isUpper(r) && (i+1 < len(field) && !isUpper(rune(field[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(toUpper(r))
	}
	return b.String()
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func toUpper(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - 'a' + 'A'
	}
	return r
}
