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

// +build windows

package dhcp4

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv4"
)

type windowsConn struct {
	conn *ipv4.PacketConn
}

func NewSnooperConn(addr string) (*Conn, error) {
	return newConn(addr, newWindowsConn)
}

func newWindowsConn(port int) (conn, error) {
	c, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	l := ipv4.NewPacketConn(c)
	return &windowsConn{l}, nil
}

func (c *windowsConn) Close() error {
	return c.conn.Close()
}

func (c *windowsConn) Recv(b []byte) (rb []byte, addr *net.UDPAddr, ifidx int, err error) {
	n, cm, a, err := c.conn.ReadFrom(b)
	if err != nil {
		return nil, nil, 0, err
	}
	return b[:n], a.(*net.UDPAddr), cm.IfIndex, nil
}

func (c *windowsConn) Send(b []byte, addr *net.UDPAddr, ifidx int) error {
	if ifidx <= 0 {
		_, err := c.conn.WriteTo(b, nil, addr)
		return err
	}
	cm := ipv4.ControlMessage{
		IfIndex: ifidx,
	}
	_, err := c.conn.WriteTo(b, &cm, addr)
	return err
}

func (c *windowsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *windowsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
