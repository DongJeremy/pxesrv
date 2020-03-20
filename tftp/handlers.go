// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tftp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
)

// FilesystemHandler returns a Handler that serves files in root.
func FilesystemHandler(root string) (Handler, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root = filepath.ToSlash(root)
	return func(path string, addr net.Addr) (io.ReadCloser, int64, error) {
		// Join with a root, which gets rid of directory traversal
		// attempts. Then we join that canonicalized path with the
		// actual root, which resolves to the actual on-disk file to
		// serve.
		path = filepath.Join("/", path)
		path = filepath.FromSlash(filepath.Join(root, path))

		st, err := os.Stat(path)
		if err != nil {
			return nil, 0, err
		}
		if !st.Mode().IsRegular() {
			return nil, 0, fmt.Errorf("requested path %q is not a file", path)
		}
		f, err := os.Open(path)
		return f, st.Size(), err
	}, nil
}

// ConstantHandler returns a Handler that serves bs for all requested paths.
func ConstantHandler(bs []byte) Handler {
	return func(path string, addr net.Addr) (io.ReadCloser, int64, error) {
		return ioutil.NopCloser(bytes.NewBuffer(bs)), int64(len(bs)), nil
	}
}
