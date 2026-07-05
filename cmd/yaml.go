/*
Copyright 2026 The BlanketOps Authors.
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

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// ----------------------
// YAML decoder
// ----------------------

var yamlDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

func decodeYAMLDocuments(reader io.Reader) ([]*unstructured.Unstructured, error) {
	all, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// split on document boundary
	parts := bytes.Split(all, []byte("\n---"))
	objs := make([]*unstructured.Unstructured, 0, len(parts))

	for _, p := range parts {
		p = bytes.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		obj := &unstructured.Unstructured{}
		_, gvk, err := yamlDecoder.Decode(p, nil, obj)
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML doc: %w", err)
		}
		if gvk == nil {
			return nil, errors.New("decoded object has nil GVK")
		}
		objs = append(objs, obj)
	}

	return objs, nil
}

// yaml stream decoder (kubectl style)
func decodeYAMLStream(reader io.Reader) ([]*unstructured.Unstructured, error) {
	dec := yamlutil.NewYAMLOrJSONDecoder(reader, 4096)
	objs := []*unstructured.Unstructured{}

	for {
		u := &unstructured.Unstructured{}
		err := dec.Decode(u)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			// ignore empty docs
			if strings.Contains(err.Error(), "no kind") || len(u.Object) == 0 {
				continue
			}

			return nil, fmt.Errorf("decoding YAML: %w", err)
		}
		if u == nil || len(u.Object) == 0 {
			continue
		}

		objs = append(objs, u)
	}

	return objs, nil
}
