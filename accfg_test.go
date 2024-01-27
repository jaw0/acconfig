// Copyright (c) 2024
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2024-Jan-26 13:21 (EST)
// Function: testing

package acconfig

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestReadLine(t *testing.T) {

	type testdata struct {
		shouldFail bool
		txt        string
		expect     []string
	}

	tests := []testdata{
		{false, "field value extra # comment\n", []string{"field", "value", "extra"}},
		{false, "field: value extra # comment\n", []string{"field", "value", "extra"}},
		{false, "field: value: extra # comment\n", []string{"field", "value:", "extra"}},
		{false, "field value: extra # comment\n", []string{"field", "value:", "extra"}},
		{false, "field: value\n", []string{"field", "value"}},
		{false, "field: value \n", []string{"field", "value"}},
		{false, "field value\n", []string{"field", "value"}},
		{false, "field value \n", []string{"field", "value"}},
		{false, "field \"value \" extra # comment\n", []string{"field", "value ", "extra"}},
	}

	for _, td := range tests {
		tok, err := tokenize(td.txt)
		if td.shouldFail {
			if err == nil {
				t.Errorf("expected to fail: '%v'", td.txt)
			}
			continue
		}
		if err != nil {
			t.Errorf("test failed '%s': %v", td.txt, err)
		}

		if err := compareSlice(tok, td.expect); err != nil {
			t.Errorf("failed. expected '%v', got '%#v': %v", td.expect, tok, err)
		}
	}
}

func compareSlice(a, b []string) error {
	if len(a) != len(b) {
		return fmt.Errorf("len mismatch %d != %d", len(a), len(b))
	}

	for i := range a {
		if a[i] != b[i] {
			return fmt.Errorf("elem %d mismatch", i)
		}
	}
	return nil
}

func tokenize(s string) ([]string, error) {
	txt := bytes.NewBufferString(s)
	fb := bufio.NewReader(txt)
	conf := conf{file: "test"}
	return conf.readLine(fb)
}

func TestReadConfig(t *testing.T) {

	txt := bytes.NewBufferString(`field value
value    123
duration 1h
doit     yes
start    2024-01-01T02:03:04Z
elapsed  1m
girth    "very very \n"

# important comment
# bool map - individually
flag slithytove
flag borogrove
flag setbutfalse off

# bool map - as block
flag {
    humpty
    dumpty   off
}

# empty map - individually
set  borogrove
set  humpty dumpty

# empty map - as block
set {
    lorem
    ipsum
}

tag  bandersnatch
tag  jubjubtree
tag  lorem ipsum dolor

thing {
    name    momerath
    size    123
}

thing {
    name    vorpalsword
}

# string map - as block
header {
    type	json
    charset     ascii
}

header {
    length	1234
}

header2 {
    type	json
    flowrate    high
}

# string map - indidually
header  refer altavista
header2 refer altavista

# struct
param {
    name    jubjubtree
    size    234
}


`)

	conf := conf{file: "test"}

	type thing struct {
		Name string
		Size int32
	}

	type stuff struct {
		Field    string
		Field2   string `accfg:"girth"`
		Value    int32
		Duration int64 `convert:"duration"`
		Doit     bool
		Flag     map[string]bool
		Set      map[string]struct{}
		Tag      []string
		Thing    []*thing
		Start    time.Time
		Elapsed  time.Duration
		Header   map[string]string
		Header2  map[string]interface{}
		Param    thing
	}

	var data stuff

	err := conf.read(txt, &data)

	if err != nil {
		t.Fatalf("readconfig failed: %v", err)
	}

	if data.Field != "value" {
		t.Errorf("read config failed: expected 'value', found %v", data.Field)
	}
	if data.Field2 != "very very \n" {
		t.Errorf("read config failed: expected 'very very \n', found %v", data.Field2)
	}
	if data.Value != 123 {
		t.Errorf("read config failed: expected '123', found %v", data.Value)
	}
	if data.Duration != 3600 {
		t.Errorf("read config failed: expected '3600', found %v", data.Duration)
	}
	if !data.Doit {
		t.Errorf("read config failed: expected 'true', found %v", data.Doit)
	}
	if !data.Flag["slithytove"] {
		t.Errorf("read config failed: failed to read flag: %+v", data)
	}
	if !data.Flag["borogrove"] {
		t.Errorf("read config failed: failed to read flag: %+v", data)
	}
	if data.Flag["missingval"] {
		t.Errorf("read config failed: failed to read flag: %+v", data)
	}
	if v, ok := data.Flag["setbutfalse"]; !ok || v {
		t.Errorf("read config failed: failed to read flag setbutfalse: %+v", data)
	}
	if _, ex := data.Set["borogrove"]; !ex {
		t.Errorf("read config failed: failed to read set: %+v", data)
	}
	if len(data.Set) != 5 {
		t.Errorf("read config failed: expected 5 elems in Set: %+v", data)
	}
	if len(data.Tag) != 5 || data.Tag[0] != "bandersnatch" {
		t.Errorf("read config failed: failed to read tag: %+v", data)
	}
	if len(data.Thing) != 2 || data.Thing[0].Name != "momerath" || data.Thing[0].Size != 123 {
		t.Errorf("read config failed: failed to read thing: %+v", data)
	}
	if data.Start.Unix() != 1704074584 {
		t.Errorf("read config failed: failed to read time: %+v", data.Start.Unix())
	}
	if data.Elapsed != time.Duration(time.Minute) {
		t.Errorf("read config failed: failed to read time: %+v", data.Elapsed)
	}
	if data.Param.Name != "jubjubtree" || data.Param.Size != 234 {
		t.Errorf("read config failed: failed to read nested struct: %+v", data.Param)
	}
	if data.Header["type"] != "json" || data.Header["length"] != "1234" {
		t.Errorf("read config failed: failed to read map: %+v", data.Header)
	}
	if data.Header["refer"] != "altavista" {
		t.Errorf("read config failed: failed to read map: %+v", data.Header)
	}
	if data.Header2["refer"] != "altavista" {
		t.Errorf("read config failed: failed to read map: %+v", data.Header2)
	}

}

func TestFail1(t *testing.T) {

	type thing struct {
		Name string
		Size int32
	}

	type stuff struct {
		Param *thing // no pointer to struct
	}

	txt := bytes.NewBufferString(`
param {
    name  slithytove
}
`)

	var data stuff

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestFail2(t *testing.T) {

	type stuff struct {
		Param map[string]int
	}

	txt := bytes.NewBufferString(`
param {
    name  slithytove
}
`)

	var data stuff

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestFail3(t *testing.T) {

	type stuff struct {
		Param *string
	}

	txt := bytes.NewBufferString(`
param lorem-ipsum
`)

	var data stuff

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestFail4(t *testing.T) {

	type stuff struct {
		Param string
	}

	txt := bytes.NewBufferString(`
param lorem-ipsum
`)

	var data []stuff

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestFail5(t *testing.T) {

	type stuff struct {
		Param string
	}

	txt := bytes.NewBufferString(`
param lorem-ipsum
`)

	var data string

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestFail6(t *testing.T) {

	type stuff struct {
		Param string
	}

	txt := bytes.NewBufferString(`
param lorem-ipsum
`)

	var data map[string]interface{}

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestFail7(t *testing.T) {

	type stuff struct {
		Param string
	}

	txt := bytes.NewBufferString(`
param lorem-ipsum
`)

	var data stuff

	conf := conf{file: "test"}
	err := conf.read(txt, data)

	if err == nil {
		t.Errorf("expected to fail: %+v", data)
	}
}

func TestArbySlice(t *testing.T) {

	type stuff struct {
		Param []int
	}

	txt := bytes.NewBufferString(`
param 123
param 234 567
`)

	var data stuff

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err != nil {
		t.Errorf("failed: %v", err)
	}
	if len(data.Param) != 3 {
		t.Errorf("failed: expected len 3, got %+v", data)
	}

	// should fail
	txt = bytes.NewBufferString(`
param {
    123
    234
}
`)
	err = conf.read(txt, &data)

	if err == nil {
		t.Errorf("expected an error")
	}

}

func TestNested(t *testing.T) {

	type common struct {
		Name string
	}
	type stuff struct {
		common
		Param string
	}

	txt := bytes.NewBufferString("name gizmo\n")

	var data stuff

	conf := conf{file: "test"}
	err := conf.read(txt, &data)

	if err != nil {
		t.Errorf("failed: %v", err)
	}
	if data.Name != "gizmo" {
		t.Errorf("failed: expected gizmo, got %+v", data)
	}

}
