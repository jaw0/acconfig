// Copyright (c) 2024
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2024-Jan-26 13:22 (EST)
// Function: AC style config files

package acconfig

import (
	"bufio"
	"encoding"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type conf struct {
	file   string
	lineNo int
	info   map[reflect.Type]fieldInfo
}

type fieldInfo map[string][]int

const DEBUG = false

// Read reads a config file into the struct
func Read(file string, cf interface{}) error {

	debugf("read %s\n", file)
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("cannot open '%s': %v", file, err)
	}

	defer f.Close()

	c := &conf{
		file:   file,
		lineNo: 1,
	}

	return c.read(f, cf)
}

func (c *conf) read(f io.Reader, cf interface{}) error {

	if err := isPtrStruct(cf); err != nil {
		return err
	}

	fb := bufio.NewReader(f)

	err := c.readConfig(fb, cf, false)
	if err != nil {
		return fmt.Errorf("cannot parse file '%s' line %d: %v", c.file, c.lineNo, err)
	}

	return nil
}

func (c *conf) learnConf(cf interface{}) fieldInfo {

	var info = make(map[string][]int)
	var val = reflect.ValueOf(cf).Elem()

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()

	// cache field info
	if c.info == nil {
		c.info = make(map[reflect.Type]fieldInfo)
	}
	if f, ok := c.info[typ]; ok {
		return f
	}

	sf := reflect.VisibleFields(typ)
	for i := range sf {
		name := strings.ToLower(sf[i].Name)
		kind := sf[i].Type.String()
		tags := sf[i].Tag

		// override default name
		if n, ok := tags.Lookup("accfg"); ok {
			name = n
		}

		info[name] = sf[i].Index
		debugf("lrn cf> %s \t%s\t%v\n", name, kind, tags)
	}

	c.info[typ] = info
	return info
}

type stringUnmarshaler interface {
	UnmarshalString(string) error
}

func (c *conf) checkAndStore(cf interface{}, info fieldInfo, k string, v string, extra []string) error {

	i, ok := info[k]
	if !ok {
		return fmt.Errorf("invalid param '%s'", k)
	}

	var cfe = reflect.ValueOf(cf).Elem()
	var cfv = cfe.FieldByIndex(i)
	var tags = cfe.Type().FieldByIndex(i).Tag

	return c.checkAndStoreField(cfv, tags, k, v, extra)
}

func (c *conf) checkAndStoreField(cfv reflect.Value, tags reflect.StructTag, k string, v string, extra []string) error {

	iv := cfv
	if iv.Kind() != reflect.Pointer && iv.Type().Name() != "" && iv.CanAddr() {
		// convert to pointer type, to find pointer methods
		iv = iv.Addr()
	}
	switch tv := iv.Interface().(type) {
	case stringUnmarshaler:
		err := tv.UnmarshalString(v)
		if err != nil {
			return fmt.Errorf("cannot parse %T for '%s': %v", tv, k, err)
		}
		return nil

	case encoding.TextUnmarshaler:
		err := tv.UnmarshalText([]byte(v))
		if err != nil {
			return fmt.Errorf("cannot parse %T for '%s': %v", tv, k, err)
		}
		return nil

		// time.Time satisfies the TextUnmarshaler interface, but Duration does not
	case *time.Duration:
		t, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("cannot parse time.Duration for '%s': %v", k, err)
		}
		cfv.Set(reflect.ValueOf(t))
		return nil

	case map[string]bool:
		if tv == nil {
			tv = make(map[string]bool)
			cfv.Set(reflect.ValueOf(tv))
		}
		if len(extra) > 0 {
			tv[v] = parseBool(extra[0])
		} else {
			tv[v] = true
		}
		return nil

	case map[string]struct{}:
		if tv == nil {
			tv = make(map[string]struct{})
			cfv.Set(reflect.ValueOf(tv))
		}
		tv[v] = struct{}{}

		// permit multiple fields: "param key1 key2 key3"
		for _, x := range extra {
			tv[x] = struct{}{}
		}
		return nil

	case map[string]string:
		if len(extra) == 0 {
			return fmt.Errorf("syntax error for %s/%s: value expected", k, v)
		}
		if tv == nil {
			tv = make(map[string]string)
			cfv.Set(reflect.ValueOf(tv))
		}
		tv[v] = extra[0]
		return nil

	case map[string]interface{}:
		if len(extra) == 0 {
			return fmt.Errorf("syntax error for %s/%s: value expected", k, v)
		}
		if tv == nil {
			tv = make(map[string]interface{})
			cfv.Set(reflect.ValueOf(tv))
		}
		tv[v] = extra[0]
		return nil

	case []string:
		tv = append(tv, v)
		// permit multiple strings: "param value1 value2 value3"
		if len(extra) > 0 {
			tv = append(tv, extra...)
		}
		cfv.Set(reflect.ValueOf(tv))
		return nil

	default:
		debugf(">> %s type %T\n", k, tv)
	}

	switch cfv.Kind() {
	case reflect.String:
		cfv.SetString(v)

	case reflect.Int, reflect.Int32, reflect.Int64:
		conv, _ := tags.Lookup("convert")
		var ix int64
		var err error

		switch conv {
		case "duration":
			ix, err = parseDuration(v)
			if err != nil {
				return fmt.Errorf("invalid value for '%s' (expected duration)\n", k)
			}
		default:
			ix, err = strconv.ParseInt(v, 0, 32)
			if err != nil {
				return fmt.Errorf("invalid value for '%s' (expected number)\n", k)
			}
		}
		cfv.SetInt(ix)

	case reflect.Float32, reflect.Float64:
		f, _ := strconv.ParseFloat(v, 64)
		cfv.SetFloat(f)

	case reflect.Bool:
		cfv.SetBool(parseBool(v))

	case reflect.Slice:
		var typ = cfv.Type().Elem()
		// permit multiple: "param value1 value2 value3"
		vals := make([]string, 0, len(extra)+1)
		vals = append(vals, v)
		vals = append(vals, extra...)

		for _, v := range vals {
			// create elem
			elem := reflect.New(typ)
			err := c.checkAndStoreField(elem.Elem(), tags, k, v, nil)
			if err != nil {
				return err
			}
			cfv.Set(reflect.Append(cfv, elem.Elem()))
		}

	default:
		return fmt.Errorf("field '%s' has unsupported type (%s)", k, cfv.Kind().String())
	}

	return nil
}

func (c *conf) readConfig(f *bufio.Reader, cf interface{}, isBlock bool) error {

	var cfinfo = c.learnConf(cf)

	for {
		tok, err := c.readLine(f)
		debugf(">> tok %v, %v", tok, err)

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if len(tok) == 0 {
			continue
		}

		key := tok[0]

		if isBlock && key == "}" {
			return nil
		}

		var val string
		var extra []string

		if len(tok) > 1 {
			val = tok[1]
		}

		if len(tok) > 2 {
			extra = tok[2:]
		}

		switch {
		case val == "{":
			err = c.readBlock(f, key, cf, cfinfo)
			if err != nil {
				return err
			}
		case key == "include":
			err = c.include(val, cf)
			if err != nil {
				return err
			}
		default:
			debugf(">>> %s => %s\n", key, val)

			err = c.checkAndStore(cf, cfinfo, key, val, extra)
			if err != nil {
				return err
			}
		}
	}
}

func (c *conf) include(file string, cf interface{}) error {
	return Read(c.includeFile(file), cf)
}

func (c *conf) includeFile(file string) string {

	if file == "" {
		return file
	}
	if file[0] == '/' {
		return file
	}

	// if file does not contain a leading path
	// make it relative to the main config file

	dir := path.Dir(c.file)
	debugf("inc dir %s, file %s\n", dir, file)

	if dir == "" {
		return file
	}

	return dir + "/" + file

}

func (c *conf) readLine(f *bufio.Reader) ([]string, error) {

	var res []string

	for {
		s, delim, err := c.readToken(f, len(res) == 0)
		debugf(">> tok %v, %v, %v\n", s, delim, err)

		if err != nil {
			return nil, err
		}

		if s != "" {
			res = append(res, s)
		}

		if delim == '\n' {
			return res, nil
		}
	}
}

func (c *conf) readToken(f *bufio.Reader, orcolon bool) (string, int, error) {
	var buf []byte

	for {
		ch, err := f.ReadByte()
		if err != nil {
			return "", -1, err
		}

		switch ch {
		case '#':
			// comment until eol
			err = eatLine(f)
			if err != nil {
				return "", -1, err
			}

			return string(buf), '\n', nil

		case '\n':
			return string(buf), '\n', nil

		case '"', '\'':
			// read until matching quote
			b, err := c.readQuoted(f, ch)
			if err != nil {
				return "", '\n', err
			}
			buf = append(buf, b...)
			continue

		case ':':
			// permit colon to delimit first token
			if !orcolon {
				break
			}
			fallthrough

		case ' ', '\t', '\r':
			if len(buf) != 0 {
				return string(buf), ' ', nil
			}
			continue
		}

		buf = append(buf, ch)
	}
}

func (c *conf) readQuoted(f *bufio.Reader, delim byte) ([]byte, error) {
	var buf []byte

	for {
		ch, err := f.ReadByte()
		if err != nil {
			return nil, err
		}
		if ch == delim {
			break
		}
		if ch == '\\' {
			// \" \' to include a quote
			ch, err = f.ReadByte()
			if err != nil {
				return nil, err
			}
			if delim == '"' {
				// within "" some \ escapes:
				switch ch {
				case 't':
					ch = '\t'
				case 'n':
					ch = '\n'
				case 'r':
					ch = '\r'
				case 'b':
					ch = '\b'
				}
			}
		}
		buf = append(buf, ch)
	}
	return buf, nil
}

func (c *conf) readMap(f *bufio.Reader, cf interface{}) error {

	for {
		tok, err := c.readLine(f)
		debugf(">> tok %v, %v", tok, err)

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if len(tok) == 0 {
			continue
		}

		key := tok[0]

		if key == "}" {
			return nil
		}

		var val string
		if len(tok) > 1 {
			val = tok[1]
		}
		debugf(">>> %s => %s\n", key, val)

		switch m := cf.(type) {
		case map[string]string:
			m[key] = val
		case map[string]interface{}:
			m[key] = val
		case map[string]bool:
			if len(tok) > 2 {
				m[key] = parseBool(tok[2])
			} else {
				m[key] = true
			}
		case map[string]struct{}:
			m[key] = struct{}{}
		default:
			return fmt.Errorf("invalid map type %T, try map[string]string", cf)
		}
	}
}

func (c *conf) readBlock(f *bufio.Reader, sect string, cf interface{}, info fieldInfo) error {

	i, ok := info[sect]
	if !ok {
		return fmt.Errorf("invalid section '%s'", sect)
	}

	var cfe = reflect.ValueOf(cf).Elem()
	var cft = cfe.Type().FieldByIndex(i).Type

	if cft.Kind() == reflect.Map {
		newcf := cfe.FieldByIndex(i)

		if newcf.IsNil() {
			// create new map
			newcf = reflect.MakeMap(cft)
			cfe.FieldByIndex(i).Set(newcf)
		}

		return c.readMap(f, newcf.Interface())
	}

	// *struct - disabled to simplify user code (no nil)
	if false && cft.Kind() == reflect.Pointer {
		s := cfe.FieldByIndex(i)
		if s.IsNil() {
			// create struct
			s = reflect.New(cft.Elem())
			cfe.FieldByIndex(i).Set(s)
		}
		return c.readConfig(f, s.Interface(), true)
	}

	if cft.Kind() == reflect.Struct {
		// nested struct
		var cfv = cfe.FieldByIndex(i).Addr().Interface()
		return c.readConfig(f, cfv, true)
	}

	// validate type is slice of pointer to struct
	if cft.Kind() != reflect.Slice || cft.Elem().Kind() != reflect.Ptr || cft.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid config type '%T'. should be []*struct, struct, or map", cf)
	}

	// create new one
	var typ = cft.Elem().Elem()
	newcf := reflect.New(typ).Interface()

	// init newcf
	// ...

	var cfv = cfe.FieldByIndex(i)
	cfv.Set(reflect.Append(cfv, reflect.ValueOf(newcf)))

	err := c.readConfig(f, newcf, true)

	return err
}

func isPtrStruct(cf interface{}) error {
	var typ = reflect.TypeOf(cf)

	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid result type, should be *struct")
	}
	return nil
}

func eatLine(f *bufio.Reader) error {
	_, _, err := f.ReadLine()
	return err
}

func debugf(txt string, args ...interface{}) {
	if DEBUG {
		fmt.Printf(txt, args...)
	}
}

// time.Duration is great for short durations (microsecs)
// but useless for real-world durations
// NB: days, months, and years are based on "typical" values and not exact
// returns seconds
func parseDuration(v string) (int64, error) {

	var lc = v[len(v)-1]
	var i int64
	var err error

	if lc >= '0' && lc <= '9' {
		i, err = strconv.ParseInt(v, 0, 32)
	} else {
		i, err = strconv.ParseInt(v[0:len(v)-1], 0, 32)

		switch unicode.ToLower(rune(lc)) {
		case 'y':
			i *= 3600 * 24 * 365
		case 'm':
			i *= 3600 * 24 * 28
		case 'd':
			i *= 3600 * 24
		case 'h':
			i *= 3600
		}

	}

	return i, err
}

func parseBool(v string) bool {

	switch strings.ToLower(v) {
	case "yes", "on", "true", "1":
		return true
	}
	return false
}
