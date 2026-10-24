module github.com/BertMarti/commitling

// The go line is also the exact Go that setup-go installs for CI, Pages and
// the Action (go-version-file). Keep it on the latest patch release: 1.27.0
// had 11 known vulnerabilities (net/http, crypto/tls, crypto/x509,
// html/template). Do not use a toolchain line instead: setup-go exports
// GOTOOLCHAIN=local, so a second use of the Action in the same job ignores it.
go 1.27.3
