import uuid

from django.db import models


class B2CTransaction(models.Model):
    """
    One row per B2C disbursement attempt. Created when we call Daraja to
    initiate the payment, then updated when the async result/timeout
    callback arrives.
    """

    class Status(models.TextChoices):
        PENDING = "PENDING", "Pending"
        ACCEPTED = "ACCEPTED", "Accepted by Safaricom"
        SUCCESS = "SUCCESS", "Success"
        FAILED = "FAILED", "Failed"
        TIMEOUT = "TIMEOUT", "Timed out"
        REJECTED = "REJECTED", "Rejected by Safaricom"

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)

    phone_number = models.CharField(max_length=15, help_text="Normalized MSISDN, e.g. 254712345678")
    amount = models.DecimalField(max_digits=12, decimal_places=2)
    remarks = models.CharField(max_length=100, blank=True)
    occasion = models.CharField(max_length=100, blank=True)
    command_id = models.CharField(max_length=50, blank=True)

    status = models.CharField(max_length=20, choices=Status.choices, default=Status.PENDING)

    # Identifiers used to correlate the initiate call with the callback.
    originator_conversation_id = models.CharField(max_length=100, unique=True)
    conversation_id = models.CharField(max_length=100, blank=True, db_index=True)

    # Populated once Safaricom accepts/rejects the initiate request.
    response_code = models.CharField(max_length=10, blank=True)
    response_description = models.CharField(max_length=255, blank=True)

    # Populated by the result callback.
    result_code = models.CharField(max_length=10, blank=True)
    result_desc = models.CharField(max_length=255, blank=True)
    transaction_id = models.CharField(max_length=50, blank=True, help_text="Safaricom M-Pesa receipt number")
    transaction_amount = models.DecimalField(max_digits=12, decimal_places=2, null=True, blank=True)
    transaction_completed_at = models.CharField(max_length=50, blank=True)
    receiver_public_name = models.CharField(max_length=255, blank=True)

    raw_initiate_response = models.JSONField(null=True, blank=True)
    raw_callback = models.JSONField(null=True, blank=True)

    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["-created_at"]

    def __str__(self):
        return f"B2C {self.id} -> {self.phone_number} ({self.status})"
