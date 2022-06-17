package twamp

import "fmt"

// Convenience function for checking the accept code contained in various TWAMP server
// response messages.
func checkAcceptStatus(accept byte, cmd string) error {
	switch accept {
	case AcceptOK:
		return nil
	case AcceptFailure:
		return fmt.Errorf("ERROR: The %s failed", cmd)
	case AcceptInternalError:
		return fmt.Errorf("ERROR: The %s failed: internal error", cmd)
	case AcceptNotSupported:
		return fmt.Errorf("ERROR: The %s failed: not supported", cmd)
	case AcceptPermResLimitation:
		return fmt.Errorf("ERROR: The %s failed: permanent resource limitation", cmd)
	case AcceptTempResLimitation:
		return fmt.Errorf("ERROR: The %s failed: temporary resource limitation", cmd)
	}
	return nil
}
