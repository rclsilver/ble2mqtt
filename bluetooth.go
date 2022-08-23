package ble2mqtt

import (
	"context"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/go-ble/ble/linux/hci/cmd"
)

type Bluetooth struct {
	device *linux.Device
}

func InitBluetooth() (*Bluetooth, error) {
	// Grab the Bluetooth LE device
	d, err := linux.NewDevice()
	if err != nil {
		return nil, err
	}

	// Reconfigure scanning to be passive
	if err := d.HCI.Send(&cmd.LESetScanParameters{
		LEScanType:           0x00,   // 0x00: passive
		LEScanInterval:       0x4000, // 0x0004 - 0x4000; N * 0.625msec
		LEScanWindow:         0x4000, // 0x0004 - 0x4000; N * 0.625msec
		OwnAddressType:       0x00,   // 0x00: public
		ScanningFilterPolicy: 0x00,   // 0x00: accept all
	}, nil); err != nil {
		return nil, err
	}

	return &Bluetooth{
		device: d,
	}, nil
}

func (bt *Bluetooth) Scan(ctx context.Context, h func(a ble.Advertisement)) error {
	if err := bt.device.Scan(ctx, true, h); err != nil && err != context.DeadlineExceeded {
		return err
	}
	return nil
}
