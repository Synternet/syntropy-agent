package twamp

// Default TCP port for remote TWAMP server.
const TwampControlPort int = 862

// Security modes for TWAMP session.
const (
	ModeUnspecified     = 0
	ModeUnauthenticated = 1 << iota
	ModeAuthenticated
	ModeEncypted
)

// TWAMP Accept Field Status Code
const (
	AcceptOK = iota
	AcceptFailure
	AcceptInternalError
	AcceptNotSupported
	AcceptPermResLimitation
	AcceptTempResLimitation
)

// TWAMP command field
// NB this field is not marked as command ir RFC, but all uses show it as that
const (
	CmdRequestSession   = 1
	CmdStartTestSession = 2
	CmdStopSessions     = 3
	CmdFetchSession     = 4
	CmdRequestTwSession = 5
)
