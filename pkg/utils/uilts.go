/*
Copyright 2023 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"bytes"
	"encoding/json"
	"strings"
)

func ToJsonString(value interface{}) (string, error) {

	jsonbyte, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(jsonbyte), nil
}

func OutputAsJson(value interface{}) string {
	jsonbyte, err := json.Marshal(value)
	if err != nil {
		return err.Error()
	}

	return string(jsonbyte)
}

func SliceToString(s []string, seq string) string {
	if s == nil || len(s) == 0 {
		return ""
	}

	buf := bytes.Buffer{}
	for _, m := range s {
		buf.WriteString(m)
		buf.WriteString(seq)
	}

	return strings.TrimSuffix(buf.String(), seq)
}

// Flatten takes a map and returns a new one where nested maps are replaced
// by dot-delimited keys.
func Flatten(m map[string]interface{}) map[string]interface{} {

	o := make(map[string]interface{})
	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nm := Flatten(child)
			for nk, nv := range nm {
				o[k+"."+nk] = nv
			}
		default:
			o[k] = v
		}
	}
	return o
}

func IsExporter(exist string, exporter []string) bool {

	for k, _ := range exporter {
		if exporter[k] == exist {
			return true
		}
	}

	return false
}

func IsExist(ss []string, key string) bool {
	for _, s := range ss {
		if s == key {
			return true
		}
	}

	return false
}

func StructToMap(data interface{}) (map[string]interface{}, error) {
	bs, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	err = json.Unmarshal(bs, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func MapToStruct(mv map[string]interface{}, v interface{}) error {
	bs, err := json.Marshal(mv)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bs, &v)
	if err != nil {
		return err
	}

	return nil
}
