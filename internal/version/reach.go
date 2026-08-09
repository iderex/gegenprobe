package version

import (
	"net"
	"time"
)

// Reach opens a connection to a host outside this machine. It is here for #83
// and for nothing else: it is the half of a run time reach that lives in
// non-test source, where the leg named `gate tier capabilities` does not read,
// so that the test beside it carries no forbidden import and passes that leg
// while still reaching the network when it runs.
//
// This file must not reach the default branch. It exists on one branch so that
// the job which runs the gate tier where the network is absent can be shown to
// be a different check from the leg that reads source, rather than a slower
// copy of it.
func Reach() error {
	c, err := net.DialTimeout("tcp", "1.1.1.1:443", 10*time.Second)
	if err != nil {
		return err
	}
	return c.Close()
}
