import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:intl/intl.dart';

import '../core/theme/app_theme.dart';

class AttendanceTimePoint {
  final DateTime date;
  final int? minutesSinceMidnight;
  final String status;

  AttendanceTimePoint({
    required this.date,
    required this.minutesSinceMidnight,
    required this.status,
  });

  static int? parseTimeOfDay(String? hms) {
    if (hms == null || hms.isEmpty) return null;
    final parts = hms.split(':');
    if (parts.length < 2) return null;
    final h = int.tryParse(parts[0]);
    final m = int.tryParse(parts[1]);
    if (h == null || m == null) return null;
    return h * 60 + m;
  }
}

/// Plots clock-in time of day against date so punctuality trends (drifting
/// later, consistently on time, etc.) are visible at a glance instead of
/// buried in a per-day list.
class AttendanceTimeChart extends StatelessWidget {
  final List<AttendanceTimePoint> points;
  final int cutoffMinutes;
  final String title;

  const AttendanceTimeChart({
    super.key,
    required this.points,
    this.cutoffMinutes = 8 * 60 + 15,
    this.title = 'Arrival Time Trend',
  });

  @override
  Widget build(BuildContext context) {
    final plotted = points.where((p) => p.minutesSinceMidnight != null).toList()
      ..sort((a, b) => a.date.compareTo(b.date));

    if (plotted.isEmpty) {
      return Container(
        height: 160,
        alignment: Alignment.center,
        child: Text(
          'No clock-in times recorded for this period',
          style: TextStyle(color: Colors.grey.shade600),
        ),
      );
    }

    final minY = (plotted.map((p) => p.minutesSinceMidnight!).reduce((a, b) => a < b ? a : b) - 15)
        .clamp(0, 24 * 60)
        .toDouble();
    final maxY = (plotted.map((p) => p.minutesSinceMidnight!).reduce((a, b) => a > b ? a : b) + 15)
        .clamp(0, 24 * 60)
        .toDouble();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Row(
            children: [
              Text(title,
                  style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 14)),
              const Spacer(),
              Icon(Icons.circle, size: 8, color: AppTheme.success),
              const SizedBox(width: 4),
              const Text('On time', style: TextStyle(fontSize: 11)),
              const SizedBox(width: 10),
              Icon(Icons.circle, size: 8, color: AppTheme.warning),
              const SizedBox(width: 4),
              const Text('Late', style: TextStyle(fontSize: 11)),
            ],
          ),
        ),
        const SizedBox(height: 10),
        SizedBox(
          height: 200,
          child: LineChart(
            LineChartData(
              minY: minY,
              maxY: maxY,
              lineTouchData: LineTouchData(
                touchTooltipData: LineTouchTooltipData(
                  getTooltipItems: (spots) => spots.map((s) {
                    final p = plotted[s.x.toInt()];
                    return LineTooltipItem(
                      '${DateFormat('MMM d').format(p.date)}\n${_fmt(p.minutesSinceMidnight!)}',
                      const TextStyle(color: Colors.white, fontWeight: FontWeight.w600),
                    );
                  }).toList(),
                ),
              ),
              gridData: FlGridData(
                show: true,
                horizontalInterval: 30,
                getDrawingHorizontalLine: (v) =>
                    FlLine(color: Colors.grey.withOpacity(0.15), strokeWidth: 1),
                drawVerticalLine: false,
              ),
              titlesData: FlTitlesData(
                topTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
                rightTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
                leftTitles: AxisTitles(
                  sideTitles: SideTitles(
                    showTitles: true,
                    reservedSize: 46,
                    interval: 30,
                    getTitlesWidget: (v, meta) => Text(
                      _fmt(v.toInt()),
                      style: const TextStyle(fontSize: 10),
                    ),
                  ),
                ),
                bottomTitles: AxisTitles(
                  sideTitles: SideTitles(
                    showTitles: true,
                    reservedSize: 24,
                    interval: (plotted.length / 5).clamp(1, plotted.length).toDouble(),
                    getTitlesWidget: (v, meta) {
                      final i = v.toInt();
                      if (i < 0 || i >= plotted.length) return const SizedBox.shrink();
                      return Padding(
                        padding: const EdgeInsets.only(top: 6),
                        child: Text(
                          DateFormat('d/M').format(plotted[i].date),
                          style: const TextStyle(fontSize: 10),
                        ),
                      );
                    },
                  ),
                ),
              ),
              borderData: FlBorderData(show: false),
              extraLinesData: ExtraLinesData(horizontalLines: [
                HorizontalLine(
                  y: cutoffMinutes.toDouble(),
                  color: AppTheme.danger.withOpacity(0.5),
                  strokeWidth: 1,
                  dashArray: [6, 4],
                  label: HorizontalLineLabel(
                    show: true,
                    alignment: Alignment.topRight,
                    style: TextStyle(color: AppTheme.danger.withOpacity(0.8), fontSize: 10),
                    labelResolver: (_) => 'Cutoff ${_fmt(cutoffMinutes)}',
                  ),
                ),
              ]),
              lineBarsData: [
                LineChartBarData(
                  spots: [
                    for (var i = 0; i < plotted.length; i++)
                      FlSpot(i.toDouble(), plotted[i].minutesSinceMidnight!.toDouble()),
                  ],
                  isCurved: false,
                  barWidth: 2,
                  color: AppTheme.primary.withOpacity(0.5),
                  dotData: FlDotData(
                    show: true,
                    getDotPainter: (spot, percent, bar, index) {
                      final late = plotted[index].minutesSinceMidnight! > cutoffMinutes;
                      return FlDotCirclePainter(
                        radius: 4,
                        color: late ? AppTheme.warning : AppTheme.success,
                        strokeWidth: 0,
                      );
                    },
                  ),
                  belowBarData: BarAreaData(show: false),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  static String _fmt(int minutes) {
    final h = minutes ~/ 60;
    final m = minutes % 60;
    final period = h >= 12 ? 'PM' : 'AM';
    final h12 = h % 12 == 0 ? 12 : h % 12;
    return '$h12:${m.toString().padLeft(2, '0')} $period';
  }
}
