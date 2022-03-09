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
