package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DamonZS/Jianmu/internal/gateway"
	"github.com/DamonZS/Jianmu/internal/mtls"
	"github.com/DamonZS/Jianmu/internal/settings"
)

const httpsUsage = `usage: jianmu https [on|off|import <cert.pem> [<key.pem>…]|remove|ca [--out <folder>]]
  other computers reach this jianmu's gateway, and jianmu web's page, at https://
  (zhangly1 on X: Claude on another computer on the LAN). This computer, and a WSL
  distro's agent, may still call plain http://.
    jianmu https              whether it is on, the certificate served and jianmu's CA
    jianmu https on|off       serve other computers over https://, or plain http:// again
    jianmu https import <files>  serve your own certificate (its chain after it) and key,
                              PEM, in one file or several; kept in jianmu's folder
    jianmu https remove       back to a certificate jianmu's own CA signs for this
                              computer's names and addresses
    jianmu https ca           save jianmu's CA as jianmu-ca.crt in Downloads (or --out)
  Without a certificate of your own, each computer that calls jianmu trusts its CA once.
  Get it from Settings, or on that computer from https://<this computer>:<port>/jianmu-ca.crt
  (check its SHA-256 against the one jianmu https prints), then:
    macOS    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain jianmu-ca.crt
    Windows  certutil -addstore -f Root jianmu-ca.crt   (as administrator)
    Linux    sudo cp jianmu-ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates
    Node apps such as Claude Code: NODE_EXTRA_CA_CERTS=/path/to/jianmu-ca.crt`

func httpsCmd(args []string) error {
	if len(args) == 0 || args[0] == "status" {
		return httpsStatus()
	}
	switch args[0] {
	case "on", "off":
		on := args[0] == "on"
		if on {
			if c, _ := gateway.ImportedFiles(); c == "" {
				if _, _, err := gateway.CA(); err != nil {
					return err
				}
			}
		}
		if err := settings.SetHTTPS(on); err != nil {
			return err
		}
		if !on {
			fmt.Println("HTTPS off: other computers call jianmu at http:// again")
			if gateway.TLSOn() {
				fmt.Println("(still served over https://: --tls-cert or mTLS is on)")
			}
			return nil
		}
		fmt.Println("HTTPS on: other computers call jianmu at https://; this computer may still call http://")
		return httpsStatus()
	case "import":
		if len(args) < 2 {
			return errors.New("usage: jianmu https import <cert.pem> [<key.pem>…]")
		}
		var files [][]byte
		for _, f := range args[1:] {
			b, err := os.ReadFile(f)
			if err != nil {
				return err
			}
			files = append(files, b)
		}
		if err := gateway.ImportCert(files...); err != nil {
			return err
		}
		fmt.Println("imported: other computers are served this certificate over https://")
		if !gateway.HTTPSOn() && !gateway.TLSOn() {
			fmt.Println("HTTPS is off: jianmu https on to serve it")
		}
		return httpsStatus()
	case "remove":
		if err := gateway.RemoveImportedCert(); err != nil {
			return err
		}
		fmt.Println("removed: jianmu's CA signs the certificate served")
		return nil
	case "ca":
		rest, out, err := outFlag(args[1:])
		if err != nil {
			return err
		}
		if len(rest) != 0 {
			return errors.New("usage: jianmu https ca [--out <folder>]")
		}
		b, fp, err := gateway.CA()
		if err != nil {
			return err
		}
		if out == "" {
			out = downloadsDir()
		}
		p := filepath.Join(out, "jianmu-ca.crt")
		if err := mtls.WriteFile(p, b, 0o644); err != nil {
			return err
		}
		fmt.Println(p)
		fmt.Println("  SHA-256 " + fp)
		return nil
	}
	return fmt.Errorf("jianmu https: unknown %q\n%s", args[0], httpsUsage)
}

func httpsStatus() error {
	state := "off"
	if gateway.HTTPSOn() {
		state = "on"
	}
	fmt.Println("HTTPS " + state)
	if !gateway.TLSOn() {
		return nil
	}
	c, err := gateway.ServedCert()
	if err != nil {
		return err
	}
	from := map[string]string{"files": "--tls-cert " + c.File, "imported": "imported", "jianmu": "jianmu's CA"}[c.From]
	fmt.Printf("  certificate  %s · %s · until %s\n", from, strings.Join(c.Names, ", "), c.Until.Format("2006-01-02"))
	if c.From == "jianmu" {
		_, fp, err := gateway.CA()
		if err != nil {
			return err
		}
		fmt.Println("  CA           " + gateway.MTLSStore().CAFile())
		fmt.Println("  SHA-256      " + fp)
		fmt.Println("  trust it on each computer that calls jianmu: jianmu https --help")
	}
	return nil
}
