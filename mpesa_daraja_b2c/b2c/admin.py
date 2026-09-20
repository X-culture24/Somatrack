from django.contrib import admin

from .models import B2CTransaction


@admin.register(B2CTransaction)
class B2CTransactionAdmin(admin.ModelAdmin):
    list_display = (
        "id",
        "phone_number",
        "amount",
        "status",
        "result_code",
        "transaction_id",
        "created_at",
    )
    list_filter = ("status", "command_id", "created_at")
    search_fields = ("phone_number", "originator_conversation_id", "conversation_id", "transaction_id")
    readonly_fields = [f.name for f in B2CTransaction._meta.fields]

    def has_add_permission(self, request):
        return False
