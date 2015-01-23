//
// Mixer for Common Multiplexing Transport Proxy (CMTP)
//
//

// TODO: delete this file

//
package cmtp

// state machine for Mixer
type MIXER_STATE int

const (
	MIXER_STATE_UNSET MIXER_STATE = iota
	MIXER_STATE_RESET
	MIXER_STATE_SENDHEADER
	MIXER_STATE_SENDBODY
	MIXER_STATE_FILLHEADER
	MIXER_STATE_FILLBODY
	MIXER_STATE_CLOSED
	MIXER_STATE_LAST
)

//
