// Copyright (c) 2024
// Author: Jeff Weisberg <tcp4me.com!jaw>
// Created: 2024-Jan-26 13:21 (EST)
// Function: testing

package acconfig

import (
	"bytes"
	"testing"
	"time"
)

func TestReadConfig(t *testing.T) {

	txt := bytes.NewBufferString(`field value
value    123
duration 1h
doit     yes
start    2024-01-01T02:03:04Z
elapsed  1m
girth    "very very \n"

# important comment
flag slithytove
flag borogrove

set  borogrove

tag  bandersnatch
tag  jubjubtree

thing {
    name    momerath
    size    123
}

thing {
    name    vorpalsword
}

header {
    type	json
}

header {
    length	1234
}

header2 {
    type	json
}

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
	if _, ex := data.Set["borogrove"]; !ex {
		t.Errorf("read config failed: failed to read set: %+v", data)
	}
	if len(data.Tag) != 2 || data.Tag[0] != "bandersnatch" {
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
