// Package e2etests holds Athens's end-to-end tests.
//
// Every functional file in this package is guarded by the "e2etests" build
// tag and is compiled only when that tag is set, e.g.
//
//	go test --tags e2etests ./e2etests
//
// (see scripts/test_e2e.sh). This file deliberately carries no build tag so
// that the package still resolves under a plain "go build ./..." or
// "go test ./..." (which then reports "no test files") instead of failing
// with "build constraints exclude all Go files in .../e2etests" -- the latter
// breaks tooling that runs tests across every package, such as the
// go-toolchain coverage run used in CI.
package e2etests
