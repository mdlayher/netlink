//+build go1.12,linux

package netlink

// This file exports certain identifiers for use in tests.

// GetThreadNetNS wraps the internal getThreadNetNS for use in tests.
func GetThreadNetNS() (int, error) { return getThreadNetNS() }

// SetThreadNetNS wraps the internal setThreadNetNS for use in tests.
func SetThreadNetNS(fd int) error { return setThreadNetNS(fd) }
