// Copyright 2017 the gousb Authors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package usb

import (
	"testing"
	"time"
)

func TestEndpoint(t *testing.T) {
	lib, done := newFakeLibusb()
	defer done()

	for _, epData := range []struct {
		ei   EndpointInfo
		intf InterfaceSetting
	}{
		{
			ei: EndpointInfo{
				Number:        2,
				Direction:     EndpointDirectionIn,
				MaxPacketSize: 512,
				TransferType:  TransferTypeBulk,
			},
			intf: InterfaceSetting{
				Number:    0,
				Alternate: 0,
				Class:     ClassVendorSpec,
			},
		},
		{
			ei: EndpointInfo{
				Number:        6,
				MaxPacketSize: 3 * 1024,
				TransferType:  TransferTypeIsochronous,
				PollInterval:  125 * time.Microsecond,
				UsageType:     IsoUsageTypeData,
			},
			intf: InterfaceSetting{
				Number:    0,
				Alternate: 0,
				Class:     ClassVendorSpec,
			},
		},
	} {
		epData.intf.Endpoints = []EndpointInfo{epData.ei}
		for _, tc := range []struct {
			desc    string
			buf     []byte
			ret     int
			status  TransferStatus
			want    int
			wantErr bool
		}{
			{
				desc: "empty buffer",
				buf:  nil,
				ret:  10,
				want: 0,
			},
			{
				desc: "128B buffer, 60 transferred",
				buf:  make([]byte, 128),
				ret:  60,
				want: 60,
			},
			{
				desc:    "128B buffer, 10 transferred and then error",
				buf:     make([]byte, 128),
				ret:     10,
				status:  TransferError,
				want:    10,
				wantErr: true,
			},
		} {
			ep := newEndpoint(nil, epData.intf, epData.ei)
			go func() {
				fakeT := lib.waitForSubmitted()
				fakeT.length = tc.ret
				fakeT.status = tc.status
				close(fakeT.done)
			}()
			got, err := ep.transfer(tc.buf)
			if (err != nil) != tc.wantErr {
				t.Errorf("%s, %s: ep.transfer(...): got err: %v, err != nil is %v, want %v", epData.ei, tc.desc, err, err != nil, tc.wantErr)
				continue
			}
			if got != tc.want {
				t.Errorf("%s, %s: ep.transfer(...): got %d bytes, want %d", epData.ei, tc.desc, got, tc.want)
			}
		}
	}
}

func TestEndpointInfo(t *testing.T) {
	for _, tc := range []struct {
		ep   EndpointInfo
		want string
	}{
		{
			ep: EndpointInfo{
				Number:        6,
				Direction:     EndpointDirectionIn,
				TransferType:  TransferTypeBulk,
				MaxPacketSize: 512,
			},
			want: "Endpoint #6 IN (address 0x86) bulk [512 bytes]",
		},
		{
			ep: EndpointInfo{
				Number:        2,
				Direction:     EndpointDirectionOut,
				TransferType:  TransferTypeIsochronous,
				MaxPacketSize: 512,
				IsoSyncType:   IsoSyncTypeAsync,
				UsageType:     IsoUsageTypeData,
			},
			want: "Endpoint #2 OUT (address 0x02) isochronous - asynchronous data [512 bytes]",
		},
		{
			ep: EndpointInfo{
				Number:        3,
				Direction:     EndpointDirectionIn,
				TransferType:  TransferTypeInterrupt,
				MaxPacketSize: 16,
				UsageType:     InterruptUsageTypePeriodic,
			},
			want: "Endpoint #3 IN (address 0x83) interrupt - periodic [16 bytes]",
		},
	} {
		if got := tc.ep.String(); got != tc.want {
			t.Errorf("%#v.String(): got %q, want %q", tc.ep, got, tc.want)
		}
	}
}

func TestEndpointIn(t *testing.T) {
	defer func(i libusbIntf) { libusb = i }(libusb)

	lib, done := newFakeLibusb()
	defer done()

	ctx := NewContext()
	defer ctx.Close()
	d, err := ctx.OpenDeviceWithVidPid(0x9999, 0x0001)
	if err != nil {
		t.Fatalf("OpenDeviceWithVidPid(0x9999, 0x0001): got error %v, want nil", err)
	}
	ep, err := d.InEndpoint(1, 0, 0, 2)
	if err != nil {
		t.Fatalf("InEndpoint(1, 0, 0, 2): got error %v, want nil", err)
	}
	dataTransferred := 100
	go func() {
		fakeT := lib.waitForSubmitted()
		fakeT.length = dataTransferred
		fakeT.status = TransferCompleted
		close(fakeT.done)
	}()
	buf := make([]byte, 512)
	got, err := ep.Read(buf)
	if err != nil {
		t.Errorf("ep.Read: got error %v, want nil", err)
	} else if got != dataTransferred {
		t.Errorf("ep.Read: got %d, want %d", got, dataTransferred)
	}

	_, err = d.InEndpoint(1, 0, 0, 1)
	if err == nil {
		t.Error("InEndpoint(1, 0, 0, 1): got nil, want error")
	}
}

func TestEndpointOut(t *testing.T) {
	defer func(i libusbIntf) { libusb = i }(libusb)

	lib, done := newFakeLibusb()
	defer done()

	ctx := NewContext()
	defer ctx.Close()
	d, err := ctx.OpenDeviceWithVidPid(0x9999, 0x0001)
	if err != nil {
		t.Fatalf("OpenDeviceWithVidPid(0x9999, 0x0001): got error %v, want nil", err)
	}
	ep, err := d.OutEndpoint(1, 0, 0, 1)
	if err != nil {
		t.Fatalf("OutEndpoint(1, 0, 0, 1): got error %v, want nil", err)
	}
	dataTransferred := 100
	go func() {
		fakeT := lib.waitForSubmitted()
		fakeT.length = dataTransferred
		fakeT.status = TransferCompleted
		close(fakeT.done)
	}()
	buf := make([]byte, 512)
	got, err := ep.Write(buf)
	if err != nil {
		t.Errorf("ep.Write: got error %v, want nil", err)
	} else if got != dataTransferred {
		t.Errorf("ep.Write: got %d, want %d", got, dataTransferred)
	}

	_, err = d.OutEndpoint(1, 0, 0, 2)
	if err == nil {
		t.Error("OutEndpoint(1, 0, 0, 2): got nil, want error")
	}
}
