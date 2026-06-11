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
