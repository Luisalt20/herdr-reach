//go:build darwin && !cgo

package probe_test

// trustStoreBuildVariant names the build variant this test binary was compiled
// in. The value is the cgo build tag's answer and nothing else: the Go
// toolchain satisfies the cgo constraint when cgo is enabled for the build, so
// this file and its cgo sibling are mutually exclusive and one of them always
// compiles into a darwin test binary. Nothing here reads the environment, the
// machine or the toolchain's defaults, because the question issue #81 asks is
// exactly which of the two binaries the release's flags produce.
const trustStoreBuildVariant = "nocgo"
