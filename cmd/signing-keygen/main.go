// signing-keygen creates the Ed25519 key pair that signs the QR codes on
// school documents. The private key file never leaves the server; point
// SIGNING_KEY_PATH at it. The printed public key is what the API publishes
// at /.well-known/stmarys-keys.json.
//
//	go run ./cmd/signing-keygen -out secrets/signing_ed25519.pem
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stmaryskabete/lms/internal/signedlink"
)

func main() {
	out := flag.String("out", "secrets/signing_ed25519.pem", "where to write the private key (refuses to overwrite)")
	flag.Parse()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	pemBytes, err := signedlink.MarshalPrivateKeyPEM(priv)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o700); err != nil {
		log.Fatal(err)
	}
	// O_EXCL: overwriting the live key would invalidate every QR code
	// already printed. Rotate by generating a new file under a new key id.
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		log.Fatalf("%v (rotate by writing a new file and bumping SIGNING_KEY_ID)", err)
	}
	if _, err := f.Write(pemBytes); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("private key: %s (mode 0600 — keep it off git and backups you don't control)\n", *out)
	fmt.Printf("public key:  %s\n", base64.RawURLEncoding.EncodeToString(pub))
	fmt.Printf("\nset in .env:\n  SIGNING_KEY_PATH=%s\n  SIGNING_KEY_ID=1\n", *out)
}
