module github.com/BertMarti/commitling

go 1.27.0

// The toolchain that CI, Pages and the Action build with (setup-go reads it
// from here, and the go command switches to it). Keep it on the latest patch
// release: 1.27.0 had 11 known vulnerabilities in net/http, crypto/tls,
// crypto/x509 and html/template that govulncheck reports.
toolchain go1.27.3
