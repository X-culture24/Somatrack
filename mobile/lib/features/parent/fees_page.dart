import 'dart:io';

import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';
import 'package:share_plus/share_plus.dart';

import '../../../core/models/school.dart';
import '../../../core/providers/attendance_provider.dart';
import '../../../core/theme/app_theme.dart';
import '../../../shared/widgets.dart';

class FeesPage extends StatefulWidget {
  const FeesPage({super.key});

  @override
  State<FeesPage> createState() => _FeesPageState();
}

class _FeesPageState extends State<FeesPage> {
  List<InvoiceSummary> _invoices = [];
  List<Student> _wards = [];
  bool _loading = true;
  String? _downloadingId;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _load());
  }

  Future<void> _load() async {
    final att = context.read<AttendanceProvider>();
    _wards = await att.myWards();
    _invoices = await att.myInvoices(force: true);
    if (mounted) setState(() => _loading = false);
  }

  double get _totalDue => _invoices.fold(0, (s, i) => s + i.totalDue);
  double get _totalPaid => _invoices.fold(0, (s, i) => s + i.totalPaid);
  double get _balance => _totalDue - _totalPaid;

  Future<void> _payFees(InvoiceSummary invoice) async {
    final ward = _wards.cast<Student?>().firstWhere((w) => w?.id == invoice.studentId, orElse: () => null);
    final amountCtrl = TextEditingController(
      text: invoice.balance > 0 ? invoice.balance.toStringAsFixed(0) : invoice.totalDue.toStringAsFixed(0),
    );
    final phoneCtrl = TextEditingController();

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Pay Fees via M-Pesa'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (ward != null) Text('For ${ward.fullName}', style: const TextStyle(fontWeight: FontWeight.w600)),
            const SizedBox(height: 12),
            TextField(
              controller: amountCtrl,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: 'Amount (KES)'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: phoneCtrl,
              keyboardType: TextInputType.phone,
              decoration: const InputDecoration(
                labelText: 'M-Pesa phone number',
                hintText: '07XXXXXXXX',
              ),
            ),
            const SizedBox(height: 8),
            Text(
              "You'll get a prompt on this phone to enter your M-Pesa PIN.",
              style: TextStyle(color: Colors.grey.shade600, fontSize: 12),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(dialogContext, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(dialogContext, true), child: const Text('Send Prompt')),
        ],
      ),
    );
    if (confirmed != true) return;

    final amount = double.tryParse(amountCtrl.text.trim());
    final phone = phoneCtrl.text.trim();
    if (amount == null || amount <= 0 || phone.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Enter a valid amount and phone number.')),
      );
      return;
    }

    if (!mounted) return;
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => const AlertDialog(
        content: Row(
          children: [
            CircularProgressIndicator(),
            SizedBox(width: 20),
            Expanded(child: Text('Sending M-Pesa prompt…')),
          ],
        ),
      ),
    );

    try {
      final att = context.read<AttendanceProvider>();
      final requestId = await att.initiatePayment(studentId: invoice.studentId, phone: phone, amount: amount);
      if (mounted) {
        Navigator.of(context, rootNavigator: true).pop(); // close "sending" dialog
        showDialog(
          context: context,
          barrierDismissible: false,
          builder: (_) => const AlertDialog(
            content: Row(
              children: [
                CircularProgressIndicator(),
                SizedBox(width: 20),
                Expanded(child: Text('Enter your M-Pesa PIN on your phone now…')),
              ],
            ),
          ),
        );
      }
      final result = await att.pollPaymentStatus(requestId);
      if (!mounted) return;
      Navigator.of(context, rootNavigator: true).pop(); // close "waiting" dialog

      final status = result['status'] as String? ?? 'pending';
      final desc = result['result_desc'] as String? ?? '';
      if (status == 'success') {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Payment received — thank you!'), backgroundColor: AppTheme.success),
        );
        _load();
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(desc.isNotEmpty ? desc : 'Payment did not complete ($status).')),
        );
      }
    } catch (e) {
      if (mounted) {
        Navigator.of(context, rootNavigator: true).pop();
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Could not start this payment: $e')),
        );
      }
    }
  }

  Future<void> _downloadStatement(InvoiceSummary invoice) async {
    setState(() => _downloadingId = invoice.id);
    try {
      final bytes = await context.read<AttendanceProvider>().invoicePdfBytes(invoice.id);
      final dir = await getTemporaryDirectory();
      final file = File('${dir.path}/invoice-${invoice.id}.pdf');
      await file.writeAsBytes(bytes, flush: true);
      if (!mounted) return;
      await Share.shareXFiles(
        [XFile(file.path, mimeType: 'application/pdf')],
        subject: 'Fee statement',
      );
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Could not download this statement: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _downloadingId = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final currency = NumberFormat.currency(locale: 'en_KE', symbol: 'KES ', decimalDigits: 0);
    return Scaffold(
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          children: [
            const SizedBox(height: 16),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Card(
                color: AppTheme.danger,
                child: Padding(
                  padding: const EdgeInsets.all(18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text(
                        'OUTSTANDING BALANCE',
                        style: TextStyle(
                            color: Colors.white70,
                            fontWeight: FontWeight.w700,
                            letterSpacing: 0.4,
                            fontSize: 12),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        currency.format(_balance),
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 34,
                          fontWeight: FontWeight.w900,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text(
                                  'Billed',
                                  style: TextStyle(
                                      color: Colors.white70, fontSize: 12),
                                ),
                                const SizedBox(height: 2),
                                Text(
                                  currency.format(_totalDue),
                                  style: const TextStyle(
                                      color: Colors.white,
                                      fontWeight: FontWeight.w700),
                                ),
                              ],
                            ),
                          ),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text(
                                  'Paid',
                                  style: TextStyle(
                                      color: Colors.white70, fontSize: 12),
                                ),
                                const SizedBox(height: 2),
                                Text(
                                  currency.format(_totalPaid),
                                  style: const TextStyle(
                                      color: Colors.white,
                                      fontWeight: FontWeight.w700),
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 12),
            const SectionHeader(title: 'Invoices'),
            if (_loading)
              const Padding(
                padding: EdgeInsets.all(40),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_invoices.isEmpty)
              const Padding(
                padding: EdgeInsets.all(40),
                child: EmptyState(
                  message: 'No invoices found for your children',
                  icon: Icons.receipt_long_outlined,
                ),
              )
            else
              ..._invoices.map(_invoiceTile).toList(),
            const SizedBox(height: 30),
          ],
        ),
      ),
    );
  }

  Widget _invoiceTile(InvoiceSummary i) {
    final currency = NumberFormat.currency(locale: 'en_KE', symbol: 'KES ', decimalDigits: 0);
    final ward = _wards.cast<Student?>().firstWhere(
          (w) => w?.id == i.studentId,
          orElse: () => null,
        );
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Card(
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () {},
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              children: [
                Row(
                  children: [
                    const Icon(Icons.receipt_long, color: AppTheme.primary),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            ward?.fullName ?? 'Student invoice',
                            style: Theme.of(context)
                                .textTheme
                                .titleSmall
                                ?.copyWith(fontWeight: FontWeight.w700),
                          ),
                          if (i.termName != null)
                            Text(
                              i.termName!,
                              style: TextStyle(
                                  color: Colors.grey.shade600, fontSize: 12),
                            ),
                        ],
                      ),
                    ),
                    StatusChip.invoice(i.status),
                  ],
                ),
                const SizedBox(height: 12),
                const Divider(height: 1),
                const SizedBox(height: 12),
                Row(
                  children: [
                    _item('Due', currency.format(i.totalDue), Colors.grey.shade700),
                    _item('Paid', currency.format(i.totalPaid), AppTheme.success),
                    _item(
                        'Balance', currency.format(i.balance),
                        i.balance > 0 ? AppTheme.danger : AppTheme.success),
                  ],
                ),
                const SizedBox(height: 10),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    if (i.balance > 0)
                      FilledButton.icon(
                        onPressed: () => _payFees(i),
                        icon: const Icon(Icons.phone_android, size: 16),
                        label: const Text('Pay via M-Pesa'),
                      ),
                    const SizedBox(width: 8),
                    OutlinedButton.icon(
                      onPressed: _downloadingId == i.id ? null : () => _downloadStatement(i),
                      icon: _downloadingId == i.id
                          ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2))
                          : const Icon(Icons.picture_as_pdf, size: 16),
                      label: Text(_downloadingId == i.id ? 'Preparing…' : 'Statement'),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _item(String label, String value, Color color) {
    return Expanded(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: TextStyle(color: Colors.grey.shade500, fontSize: 11)),
          const SizedBox(height: 2),
          Text(
            value,
            style: TextStyle(fontWeight: FontWeight.w700, color: color),
          ),
        ],
      ),
    );
  }
}
