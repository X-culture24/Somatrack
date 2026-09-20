"""
Helpers for talking to Safaricom's Daraja API that don't belong on the
client class itself: phone-number normalisation and the RSA encryption
Safaricom requires for the B2C "SecurityCredential" field.
"""
import base64
import re

from cryptography.hazmat.primitives.asymmetric import padding
from cryptography.x509 import load_der_x509_certificate, load_pem_x509_certificate


class DarajaConfigError(Exception):
    """Raised when required Daraja configuration is missing or invalid."""


def normalize_msisdn(phone_number: str) -> str:
    """
    Normalize a Kenyan phone number to the 2547XXXXXXXX / 2541XXXXXXXX
    format Daraja expects.

    Accepts formats like: 0712345678, 712345678, +254712345678, 254712345678.
    """
    digits = re.sub(r"\D", "", phone_number or "")

    if digits.startswith("254") and len(digits) == 12:
        return digits
    if digits.startswith("0") and len(digits) == 10:
        return "254" + digits[1:]
    if len(digits) == 9 and digits[0] in ("7", "1"):
        return "254" + digits

    raise ValueError(f"'{phone_number}' is not a recognisable Safaricom MSISDN")


def generate_security_credential(initiator_password: str, certificate_path: str) -> str:
    """
    Encrypt the initiator password with Safaricom's public certificate as
    required for the B2C `SecurityCredential` field.

    Safaricom publish the certificate as an X.509 file (PEM or DER encoded
    depending on where you download it from). We try PEM first, then DER,
    encrypt with RSA/PKCS1v15 (per Daraja docs) and base64-encode the result.
    """
    if not initiator_password:
        raise DarajaConfigError(
            "DARAJA_INITIATOR_PASSWORD is not set — cannot derive SecurityCredential."
        )

    try:
        with open(certificate_path, "rb") as cert_file:
            cert_bytes = cert_file.read()
    except OSError as exc:
        raise DarajaConfigError(
            f"Could not read Daraja public certificate at '{certificate_path}': {exc}"
        ) from exc

    try:
        certificate = load_pem_x509_certificate(cert_bytes)
    except ValueError:
        certificate = load_der_x509_certificate(cert_bytes)

    public_key = certificate.public_key()
    encrypted = public_key.encrypt(initiator_password.encode("utf-8"), padding.PKCS1v15())
    return base64.b64encode(encrypted).decode("utf-8")
