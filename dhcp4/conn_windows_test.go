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

//+build windows

package dhcp4

import (
	"net"
	"runtime"
	"testing"
)

func TestWindowsConn(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("not supported on %s", runtime.GOOS)
	}

	// Use a listener to grab a free port, but we don't use it beyond
	// that.
	l, err := net.ListenPacket("udp4", "127.0.0.1:69")
	if err != nil {
		t.Fatal(err)
	}
	c, err := newWindowsConn(l.LocalAddr().(*net.UDPAddr).Port)
	if err != nil {
		t.Fatalf("creating the windowsconn: %s", err)
	}

	testConn(t, c, l.LocalAddr().String())
}
