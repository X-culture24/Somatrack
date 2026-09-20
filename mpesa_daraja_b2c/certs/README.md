# Daraja public certificates

Place Safaricom's public certificate here so `DARAJA_CERTIFICATE_PATH` can
find it. It is used to RSA-encrypt the initiator password into the
`SecurityCredential` field required by the B2C API.

- **Sandbox**: download `SandboxCertificate.cer` from the Daraja docs'
  "B2C" / "Generate Security Credential" page:
  https://developer.safaricom.co.ke/Documentation
- **Production**: Safaricom provides `ProductionCertificate.cer` as part of
  your go-live onboarding — request it from your Safaricom account manager.

Do not commit real certificates or the `.env` file containing your
initiator password to a public repository.
