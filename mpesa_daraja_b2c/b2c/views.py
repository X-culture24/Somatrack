import json
import logging

from django.conf import settings
from django.utils.decorators import method_decorator
from django.views.decorators.csrf import csrf_exempt
from rest_framework import status
from rest_framework.permissions import AllowAny
from rest_framework.response import Response
from rest_framework.views import APIView

from .models import B2CTransaction
from .serializers import B2CPaymentRequestSerializer, B2CTransactionSerializer
from .services import DarajaAPIError, DarajaB2CClient

logger = logging.getLogger("b2c")


def _check_shared_secret(request):
    """Optional lightweight auth for the initiate endpoint (see .env.example)."""
    expected = settings.DARAJA_API_SHARED_SECRET
    if not expected:
        return True
    provided = request.headers.get("X-API-KEY", "")
    return provided == expected


class InitiateB2CPaymentView(APIView):
    """
    POST /api/b2c/initiate/
    {
        "phone_number": "0712345678",
        "amount": 100,
        "remarks": "Refund for order #123",
        "occasion": "Order #123"
    }

    Creates a PENDING B2CTransaction, asks Daraja to process the payout, and
    returns Safaricom's acknowledgement. The final outcome (success/failure)
    arrives asynchronously on the result callback and updates this record.
    """

    permission_classes = [AllowAny]

    def post(self, request):
        if not _check_shared_secret(request):
            return Response({"detail": "Invalid or missing API key."}, status=status.HTTP_401_UNAUTHORIZED)

        serializer = B2CPaymentRequestSerializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        data = serializer.validated_data

        client = DarajaB2CClient()
        try:
            daraja_response = client.send_payment(
                phone_number=data["phone_number"],
                amount=data["amount"],
                remarks=data.get("remarks", "B2C Payment"),
                occasion=data.get("occasion", ""),
                command_id=data.get("command_id"),
            )
        except DarajaAPIError as exc:
            logger.exception("B2C initiate failed")
            return Response(
                {"detail": str(exc)}, status=status.HTTP_502_BAD_GATEWAY
            )
        except ValueError as exc:
            # e.g. bad phone number from normalize_msisdn
            return Response({"detail": str(exc)}, status=status.HTTP_400_BAD_REQUEST)

        transaction = B2CTransaction.objects.create(
            phone_number=self._safe_msisdn(data["phone_number"]),
            amount=data["amount"],
            remarks=data.get("remarks", "B2C Payment"),
            occasion=data.get("occasion", ""),
            command_id=data.get("command_id") or settings.DARAJA_B2C_COMMAND_ID,
            originator_conversation_id=daraja_response.get("_originator_conversation_id", ""),
            conversation_id=daraja_response.get("ConversationID", ""),
            response_code=str(daraja_response.get("ResponseCode", "")),
            response_description=daraja_response.get("ResponseDescription", ""),
            status=(
                B2CTransaction.Status.ACCEPTED
                if str(daraja_response.get("ResponseCode")) == "0"
                else B2CTransaction.Status.REJECTED
            ),
            raw_initiate_response=daraja_response,
        )

        return Response(
            B2CTransactionSerializer(transaction).data,
            status=status.HTTP_202_ACCEPTED,
        )

    @staticmethod
    def _safe_msisdn(raw_phone):
        from .utils import normalize_msisdn

        try:
            return normalize_msisdn(raw_phone)
        except ValueError:
            return raw_phone


class B2CTransactionDetailView(APIView):
    """GET /api/b2c/transactions/<uuid:pk>/ — poll a transaction's status."""

    permission_classes = [AllowAny]

    def get(self, request, pk):
        try:
            transaction = B2CTransaction.objects.get(pk=pk)
        except B2CTransaction.DoesNotExist:
            return Response({"detail": "Not found."}, status=status.HTTP_404_NOT_FOUND)
        return Response(B2CTransactionSerializer(transaction).data)


def _extract_result_parameters(result: dict) -> dict:
    """Flatten Daraja's ResultParameter list into a plain dict."""
    params = {}
    result_params = (result.get("ResultParameters") or {}).get("ResultParameter") or []
    for item in result_params:
        key = item.get("Key")
        if key:
            params[key] = item.get("Value")
    return params


@method_decorator(csrf_exempt, name="dispatch")
class B2CResultCallbackView(APIView):
    """
    POST /api/b2c/callback/result/

    Safaricom posts the final outcome of a B2C request here, asynchronously,
    some time after the initiate call returned. This URL must be a publicly
    reachable HTTPS endpoint (registered as DARAJA_B2C_RESULT_URL / passed
    to Safaricom support for whitelisting on sandbox).
    """

    permission_classes = [AllowAny]
    authentication_classes = []

    def post(self, request):
        payload = request.data or json.loads(request.body or "{}")
        logger.info("B2C result callback received: %s", payload)

        result = payload.get("Result", {})
        originator_conversation_id = result.get("OriginatorConversationID", "")
        conversation_id = result.get("ConversationID", "")
        result_code = result.get("ResultCode")

        transaction = self._find_transaction(originator_conversation_id, conversation_id)
        if transaction is None:
            logger.warning(
                "B2C result callback for unknown transaction (originator=%s, conversation=%s)",
                originator_conversation_id,
                conversation_id,
            )
            # Still ack with ResultCode 0 so Safaricom doesn't retry forever.
            return self._ack()

        params = _extract_result_parameters(result)
        transaction.raw_callback = payload
        transaction.result_code = str(result_code) if result_code is not None else ""
        transaction.result_desc = result.get("ResultDesc", "")
        transaction.conversation_id = conversation_id or transaction.conversation_id
        transaction.transaction_id = result.get("TransactionID", "") or transaction.transaction_id
        transaction.transaction_amount = params.get("TransactionAmount") or transaction.transaction_amount
        transaction.transaction_completed_at = params.get("TransactionCompletedDateTime", "")
        transaction.receiver_public_name = params.get("ReceiverPartyPublicName", "")
        transaction.status = (
            B2CTransaction.Status.SUCCESS if str(result_code) == "0" else B2CTransaction.Status.FAILED
        )
        transaction.save()

        return self._ack()

    def _find_transaction(self, originator_conversation_id, conversation_id):
        qs = B2CTransaction.objects
        if originator_conversation_id:
            match = qs.filter(originator_conversation_id=originator_conversation_id).first()
            if match:
                return match
        if conversation_id:
            return qs.filter(conversation_id=conversation_id).first()
        return None

    @staticmethod
    def _ack():
        # Safaricom expects this exact envelope to consider the callback delivered.
        return Response({"ResultCode": 0, "ResultDesc": "Callback received successfully"})


@method_decorator(csrf_exempt, name="dispatch")
class B2CTimeoutCallbackView(APIView):
    """
    POST /api/b2c/callback/timeout/

    Safaricom posts here if the transaction request itself times out
    (distinct from the payment failing) before a result is produced.
    """

    permission_classes = [AllowAny]
    authentication_classes = []

    def post(self, request):
        payload = request.data or json.loads(request.body or "{}")
        logger.warning("B2C timeout callback received: %s", payload)

        result = payload.get("Result", payload)
        originator_conversation_id = result.get("OriginatorConversationID", "")
        conversation_id = result.get("ConversationID", "")

        transaction = B2CTransaction.objects.filter(
            originator_conversation_id=originator_conversation_id
        ).first() or B2CTransaction.objects.filter(conversation_id=conversation_id).first()

        if transaction:
            transaction.status = B2CTransaction.Status.TIMEOUT
            transaction.raw_callback = payload
            transaction.result_desc = result.get("ResultDesc", "Request timed out")
            transaction.save()

        return Response({"ResultCode": 0, "ResultDesc": "Callback received successfully"})
