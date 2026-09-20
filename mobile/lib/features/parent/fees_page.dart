import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

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
            const SectionHeader(title: 'Payment Actions'),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () {},
                      icon: const Icon(Icons.phone_android),
                      label: const Text('M-Pesa Pay'),
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: () {},
                      icon: const Icon(Icons.picture_as_pdf),
                      label: const Text('Statement'),
                    ),
                  ),
                ],
              ),
            ),
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
