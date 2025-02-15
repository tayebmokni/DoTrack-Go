// Package gt06 implements decoders for the GT06 GPS protocol
package gt06

// Protocol constants
const (
	startByte1 = 0x78
	startByte2 = 0x78
	endByte1   = 0x0D
	endByte2   = 0x0A

	// Message types
	loginMsg    = 0x01
	locationMsg = 0x12
	statusMsg   = 0x13
	alarmMsg    = 0x16

	// Alarm types
	sosAlarm        = 0x01
	powerCutAlarm   = 0x02
	vibrationAlarm  = 0x03
	fenceInAlarm    = 0x04
	fenceOutAlarm   = 0x05
	lowBatteryAlarm = 0x06
	overspeedAlarm  = 0x07
)
