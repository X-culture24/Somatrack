package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	phoneRe = regexp.MustCompile(`^\+?[0-9]{9,15}$`)
	timeRe  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`)
)

var errEmpty = errors.New("is required")

// convert turns one decoded JSON value (string, json.Number, bool or nil) or
// a raw query-string value into the Go value bound for f's column. A nil
// result means SQL NULL.
func convert(f *Field, v interface{}) (interface{}, error) {
	if v == nil {
		return nil, nil
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			// Empty text stays "" (columns are NOT NULL DEFAULT ''); empty
			// anything-else is "not given".
			if isText(f.Type) && f.Type != Enum {
				return "", nil
			}
			return nil, nil
		}
		v = s
	}

	switch f.Type {
	case Text, LongText, Phone, Email, Enum:
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("must be text")
		}
		if f.MaxLen > 0 && len([]rune(s)) > f.MaxLen {
			return nil, fmt.Errorf("must be at most %d characters", f.MaxLen)
		}
		switch f.Type {
		case Email:
			if !emailRe.MatchString(s) {
				return nil, errors.New("must be a valid email address")
			}
		case Phone:
			compact := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(s)
			if !phoneRe.MatchString(compact) {
				return nil, errors.New("must be a valid phone number")
			}
			s = compact
		case Enum:
			if !contains(f.Options, s) {
				return nil, fmt.Errorf("must be one of: %s", strings.Join(f.Options, ", "))
			}
		}
		return s, nil

	case Int:
		n, err := number(v)
		if err != nil {
			return nil, err
		}
		if n != math.Trunc(n) {
			return nil, errors.New("must be a whole number")
		}
		if err := bounds(f, n); err != nil {
			return nil, err
		}
		return int64(n), nil

	case Decimal:
		n, err := number(v)
		if err != nil {
			return nil, err
		}
		if err := bounds(f, n); err != nil {
			return nil, err
		}
		return n, nil

	case Bool:
		switch b := v.(type) {
		case bool:
			return b, nil
		case string:
			p, err := strconv.ParseBool(b)
			if err != nil {
				return nil, errors.New("must be true or false")
			}
			return p, nil
		}
		return nil, errors.New("must be true or false")

	case Date:
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("must be a date (YYYY-MM-DD)")
		}
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil, errors.New("must be a date (YYYY-MM-DD)")
		}
		return d, nil

	case DateTime:
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("must be a date and time")
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			// <input type="datetime-local"> sends no zone; the school is
			// in Nairobi.
			t, err = time.ParseInLocation("2006-01-02T15:04", s, nairobi)
		}
		if err != nil {
			return nil, errors.New("must be a date and time")
		}
		return t, nil

	case Time:
		s, ok := v.(string)
		if !ok || !timeRe.MatchString(s) {
			return nil, errors.New("must be a time (HH:MM)")
		}
		return s, nil

	case Ref:
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("must be an id")
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, errors.New("must be a valid id")
		}
		return id, nil
	}
	return nil, fmt.Errorf("unsupported type %s", f.Type)
}

var nairobi = func() *time.Location {
	if l, err := time.LoadLocation("Africa/Nairobi"); err == nil {
		return l
	}
	return time.FixedZone("EAT", 3*3600)
}()

func number(v interface{}) (float64, error) {
	switch n := v.(type) {
	case json.Number:
		return n.Float64()
	case float64:
		return n, nil
	case string:
		f, err := strconv.ParseFloat(strings.ReplaceAll(n, ",", ""), 64)
		if err != nil {
			return 0, errors.New("must be a number")
		}
		return f, nil
	}
	return 0, errors.New("must be a number")
}

func bounds(f *Field, n float64) error {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return errors.New("must be a number")
	}
	if f.Min != nil && n < *f.Min {
		return fmt.Errorf("must be at least %v", *f.Min)
	}
	if f.Max != nil && n > *f.Max {
		return fmt.Errorf("must be at most %v", *f.Max)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// selectExpr is how a field is read back, normalised to JSON-friendly Go
// types (dates and ids as text, numerics as float).
func selectExpr(f *Field) string {
	col := "t." + f.Name
	switch f.Type {
	case Ref:
		return col + "::text"
	case Date:
		return col + "::text"
	case Time:
		return "to_char(" + col + ", 'HH24:MI')"
	case Decimal:
		return col + "::float8"
	case Int:
		return col + "::bigint"
	}
	return col
}

// placeholder is the bind parameter for f, with the cast its column needs.
func placeholder(f *Field, n int) string {
	p := "$" + strconv.Itoa(n)
	if f.Type == Time {
		return p + "::time"
	}
	return p
}
