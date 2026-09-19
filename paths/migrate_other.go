//go:build !windows

package paths

func isCaseInsensitivePathPlatform() bool { return false }
