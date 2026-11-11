package cli

import (
	"fmt"
	"strings"

	"github.com/DamonZS/Jianmu/internal/gateway"
)

// tlsFlags takes --tls-cert <file> and --tls-key <file> (or =<file>) out
// of args, sets them for the gateway over JIANMU_TLS_CERT and
// JIANMU_TLS_KEY, and checks the pair: one without the other, or files
// that don't load, refuse to start rather than serve plain HTTP the user
// didn't ask for. rest is the other arguments.
func tlsFlags(args []string) (rest []string, err error) {
	cert, key := gateway.TLSFiles()
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--tls-cert" || a == "--tls-key":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return nil, fmt.Errorf("%s takes a file: %s <file>", a, a)
			}
			i++
			if a == "--tls-cert" {
				cert = args[i]
			} else {
				key = args[i]
			}
		case strings.HasPrefix(a, "--tls-cert="):
			cert = strings.TrimPrefix(a, "--tls-cert=")
		case strings.HasPrefix(a, "--tls-key="):
			key = strings.TrimPrefix(a, "--tls-key=")
		default:
			rest = append(rest, a)
		}
	}
	return rest, gateway.SetTLS(cert, key)
}

// tlsLine is the banner's line for TLS, "" with it off.
func tlsLine() string {
	if !gateway.TLSOn() {
		return ""
	}
	cert, _ := gateway.TLSFiles()
	if cert == "" {
		cert = "jianmu's own CA"
		if c, _ := gateway.ImportedFiles(); c != "" {
			cert = c
		}
	}
	s := muted.Render("  TLS     ") + " " + cert
	if sum := gateway.TLSSummary(); sum != "" {
		s += muted.Render(" · " + sum)
	}
	if gateway.MTLSOn() {
		s += "\n" + muted.Render("           other computers show a client certificate jianmu issued (jianmu mtls)")
	}
	return s + "\n" + muted.Render("           https:// from other computers, read again when the files change; this computer may call plain http://")
}
