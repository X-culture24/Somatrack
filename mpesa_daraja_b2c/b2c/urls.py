from django.urls import path

from . import views

app_name = "b2c"

urlpatterns = [
    path("initiate/", views.InitiateB2CPaymentView.as_view(), name="initiate"),
    path("transactions/<uuid:pk>/", views.B2CTransactionDetailView.as_view(), name="transaction-detail"),
    path("callback/result/", views.B2CResultCallbackView.as_view(), name="result-callback"),
    path("callback/timeout/", views.B2CTimeoutCallbackView.as_view(), name="timeout-callback"),
]
