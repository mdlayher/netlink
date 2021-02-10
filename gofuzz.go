//+build gofuzz

package netlink

func Fuzz(b []byte) int { return fuzz(b) }
