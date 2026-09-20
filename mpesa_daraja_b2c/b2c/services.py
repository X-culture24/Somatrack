"""
Thin client around Safaricom's Daraja B2C API.

Usage:
    from b2c.services import DarajaB2CClient

    client = DarajaB2CClient()
    response = client.send_payment(
        phone_number="0712345678",
        amount=100,
        remarks="Refund",
        occasion="Order #123",
    )
"""
import logging
import uuid

import requests
from django.conf import settings
from django.core.cache import cache

from .utils import generate_security_credential, normalize_msisdn

logger = logging.getLogger("b2c")

ACCESS_TOKEN_CACHE_KEY = "daraja:b2c:access_token"
# Safaricom tokens are valid for 3600s; refresh a little early to be safe.
TOKEN_SAFETY_MARGIN_SECONDS = 60


class DarajaAPIError(Exception):
    """Raised when Daraja returns an error response or the request fails."""

    def __init__(self, message, response=None):
        super().__init__(message)
        self.response = response


class DarajaB2CClient:
    """Handles OAuth token acquisition and B2C payment requests."""

    def __init__(self):
        self.base_url = settings.DARAJA_BASE_URL
        self.consumer_key = settings.DARAJA_CONSUMER_KEY
        self.consumer_secret = settings.DARAJA_CONSUMER_SECRET
        self.timeout = 30

    # -- OAuth ---------------------------------------------------------

    def get_access_token(self, force_refresh: bool = False) -> str:
        """Return a cached OAuth access token, fetching a new one if needed."""
        if not force_refresh:
            cached = cache.get(ACCESS_TOKEN_CACHE_KEY)
            if cached:
                return cached

        if not self.consumer_key or not self.consumer_secret:
            raise DarajaAPIError(
                "DARAJA_CONSUMER_KEY / DARAJA_CONSUMER_SECRET are not configured."
            )

        url = f"{self.base_url}/oauth/v1/generate?grant_type=client_credentials"
        try:
            resp = requests.get(
                url,
                auth=(self.consumer_key, self.consumer_secret),
                timeout=self.timeout,
            )
        except requests.RequestException as exc:
            raise DarajaAPIError(f"Failed to reach Daraja OAuth endpoint: {exc}") from exc

        if resp.status_code != 200:
            raise DarajaAPIError(
                f"Daraja OAuth request failed ({resp.status_code}): {resp.text}", response=resp
            )

        data = resp.json()
        token = data.get("access_token")
        if not token:
            raise DarajaAPIError(f"Daraja OAuth response missing access_token: {data}")

        expires_in = int(data.get("expires_in", 3600))
        cache.set(
            ACCESS_TOKEN_CACHE_KEY,
            token,
            timeout=max(expires_in - TOKEN_SAFETY_MARGIN_SECONDS, 30),
        )
        return token

    # -- B2C -------------------------------------------------------------

    def _security_credential(self) -> str:
        return generate_security_credential(
            settings.DARAJA_INITIATOR_PASSWORD, settings.DARAJA_CERTIFICATE_PATH
        )

    def send_payment(
        self,
        phone_number: str,
        amount,
        remarks: str,
        occasion: str = "",
        command_id: str = None,
        originator_conversation_id: str = None,
    ) -> dict:
        """
        Trigger a B2C payment via Safaricom's /mpesa/b2c/v1/paymentrequest
        endpoint and return Safaricom's JSON response (which contains the
        ConversationID / OriginatorConversationID to track the async result).

        Actual success/failure of the payment arrives later on the
        result callback URL — this call only confirms Safaricom *accepted*
        the request for processing.
        """
        msisdn = normalize_msisdn(phone_number)
        originator_conversation_id = originator_conversation_id or str(uuid.uuid4())

        payload = {
            "OriginatorConversationID": originator_conversation_id,
            "InitiatorName": settings.DARAJA_INITIATOR_NAME,
            "SecurityCredential": self._security_credential(),
            "CommandID": command_id or settings.DARAJA_B2C_COMMAND_ID,
            "Amount": str(int(amount)),
            "PartyA": settings.DARAJA_INITIATOR_SHORTCODE,
            "PartyB": msisdn,
            "Remarks": remarks[:100] if remarks else "B2C Payment",
            "QueueTimeOutURL": settings.DARAJA_B2C_TIMEOUT_URL,
            "ResultURL": settings.DARAJA_B2C_RESULT_URL,
            "Occasion": (occasion or "")[:100],
        }

        response_json = self._post(
            "/mpesa/b2c/v1/paymentrequest", payload, retry_on_auth_failure=True
        )
        response_json["_originator_conversation_id"] = originator_conversation_id
        return response_json

    # -- internals ---------------------------------------------------------

    def _post(self, path: str, payload: dict, retry_on_auth_failure: bool = False) -> dict:
        token = self.get_access_token()
        url = f"{self.base_url}{path}"
        headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

        try:
            resp = requests.post(url, json=payload, headers=headers, timeout=self.timeout)
        except requests.RequestException as exc:
            raise DarajaAPIError(f"Failed to reach Daraja at {path}: {exc}") from exc

        if resp.status_code == 401 and retry_on_auth_failure:
            logger.warning("Daraja access token rejected, refreshing and retrying once.")
            token = self.get_access_token(force_refresh=True)
            headers["Authorization"] = f"Bearer {token}"
            try:
                resp = requests.post(url, json=payload, headers=headers, timeout=self.timeout)
            except requests.RequestException as exc:
                raise DarajaAPIError(f"Failed to reach Daraja at {path}: {exc}") from exc

        try:
            data = resp.json()
        except ValueError:
            data = {"raw": resp.text}

        if resp.status_code >= 400:
            logger.error("Daraja request to %s failed (%s): %s", path, resp.status_code, data)
            raise DarajaAPIError(
                f"Daraja request failed ({resp.status_code}): {data}", response=resp
            )

        return data
