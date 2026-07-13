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
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/blanketops/environments-cli/util"
)

// ---------------------------------------------------------------------------
// UninstallManifests
// ---------------------------------------------------------------------------
func UninstallManifests(manifestPaths []string) error {
	dc, cfg, err := NewDynamicClient()
	if err != nil {
		return err
	}

	mapper, err := NewRESTMapper(cfg)
	if err != nil {
		return err
	}

	for _, path := range manifestPaths {
		p := util.NewSpinner(path)

		data, err := os.ReadFile(path)
		if err != nil {
			err = fmt.Errorf("read manifest %s: %w", path, err)
			p.Fail(err)
			return err
		}

		objs, err := decodeYAMLStream(bytes.NewReader(data))
		if err != nil {
			err = fmt.Errorf("decode %s: %w", path, err)
			p.Fail(err)
			return err
		}

		for _, o := range objs {
			gvk := o.GroupVersionKind()
			p.Update(fmt.Sprintf("%s %s", gvk.Kind, o.GetName()))

			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				continue
			}

			var ri dynamic.ResourceInterface
			if mapping.Scope.Name() == "namespace" {
				ns := o.GetNamespace()
				if ns == "" {
					ns = "default"
				}
				ri = dc.Resource(mapping.Resource).Namespace(ns)
			} else {
				ri = dc.Resource(mapping.Resource)
			}

			_ = ri.Delete(context.Background(), o.GetName(), metav1.DeleteOptions{})
		}
		p.Done("")
	}

	return nil
}
