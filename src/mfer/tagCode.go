package mfer

const (
	// for Mfer.Sampling
	INTERVAL    = 0x0b
	SENSITIVITY = 0x0c
	DATA_TYPE   = 0x0a
	OFFSET      = 0x0d
	NULL        = 0x12

	// for Mfer.Frame
	BLOCK     = 0x04
	CHANNEL   = 0x05
	SEQUENCE  = 0x06
	F_POINTER = 0x07

	// for Mfer.WaveForm
	WAVE_FORM_TYPE    = 0x08
	CHANNEL_ATTRIBUTE = 0x3f
	LDN               = 0x09
	INFORMATION       = 0x15
	FILTER            = 0x11
	IPD               = 0x0f
	DATA              = 0x1e

	// for Mfer.Control
	BYTE_ORDER   = 0x01
	VERSION      = 0x02
	CHAR_CODE    = 0x03
	ZERO         = 0x00
	COMMENT      = 0x16
	MACHINE_INFO = 0x17
	COMPRESSION  = 0x0e

	// for extensions
	PREAMBLE  = 0x40
	EVENT     = 0x41
	VALUE     = 0x42
	CONDITION = 0x44
	ERROR     = 0x43
	GROUP     = 0x67
	R_POINTER = 0x45
	SIGNITURE = 0x46

	// for Mfer.Helper
	P_NAME  = 0x81
	P_ID    = 0x82
	P_AGE   = 0x83
	P_SEX   = 0x84
	TIME    = 0x85
	MESSAGE = 0x86
	UID     = 0x87
	MAP     = 0x88
	END     = 0x80
)
