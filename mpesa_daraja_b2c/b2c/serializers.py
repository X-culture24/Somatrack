from rest_framework import serializers

from .models import B2CTransaction


class B2CPaymentRequestSerializer(serializers.Serializer):
    """Input payload for POST /api/b2c/initiate/"""

    phone_number = serializers.CharField(max_length=15)
    amount = serializers.DecimalField(max_digits=12, decimal_places=2, min_value=1)
    remarks = serializers.CharField(max_length=100, required=False, default="B2C Payment")
    occasion = serializers.CharField(max_length=100, required=False, allow_blank=True, default="")
    command_id = serializers.ChoiceField(
        choices=["SalaryPayment", "BusinessPayment", "PromotionPayment"],
        required=False,
    )


class B2CTransactionSerializer(serializers.ModelSerializer):
    class Meta:
        model = B2CTransaction
        fields = [
            "id",
            "phone_number",
            "amount",
            "remarks",
            "occasion",
            "command_id",
            "status",
            "originator_conversation_id",
            "conversation_id",
            "response_code",
            "response_description",
            "result_code",
            "result_desc",
            "transaction_id",
            "transaction_amount",
            "transaction_completed_at",
            "receiver_public_name",
            "created_at",
            "updated_at",
        ]
        read_only_fields = fields
