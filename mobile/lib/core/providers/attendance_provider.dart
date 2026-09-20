import 'package:flutter/foundation.dart';
import 'package:intl/intl.dart';

import '../api/api_client.dart';
import '../models/school.dart';
import 'auth_provider.dart';

class AttendanceProvider extends ChangeNotifier {
  final ApiClient _api = ApiClient();
  AuthProvider? _auth;

  Map<String, DailyAttendanceSummary> _dailyCache = {};
  Map<String, StudentAttendanceRange> _studentRangeCache = {};
  Map<String, StaffAttendanceRange> _staffRangeCache = {};
  Map<String, List<ClassRosterAttendance>> _classRosterCache = {};
  Map<String, List<Student>> _wardCache = {};
  Map<String, List<Announcement>> _announcementsCache = {};
  Map<String, List<InvoiceSummary>> _invoiceCache = {};
  List<ClassGroup> _myClasses = [];

  bool _loading = false;
  String? _error;

  bool get loading => _loading;
  String? get error => _error;
  List<ClassGroup> get myClasses => _myClasses;

  void updateAuth(AuthProvider auth) {
    _auth = auth;
    _api.updateToken(auth.accessToken);
  }

  void _setLoading(bool v) {
    _loading = v;
    notifyListeners();
  }

  Future<DailyAttendanceSummary?> dailySummary({
    String? classId,
    DateTime? date,
  }) async {
    date ??= DateTime.now();
    final dateStr = DateFormat('yyyy-MM-dd').format(date);
    final key = '${classId ?? 'all'}:$dateStr';
    if (_dailyCache.containsKey(key)) return _dailyCache[key];
    try {
      final res = await _api.get<DailyAttendanceSummary>(
        '/attendance/daily-summary/',
        query: {
          if (classId != null) 'class_id': classId,
          'date': dateStr,
        },
        decode: (d) => d is Map
            ? DailyAttendanceSummary.fromJson(d as Map<String, dynamic>)
            : DailyAttendanceSummary(
                date: dateStr,
                presentCount: 0,
                lateCount: 0,
                absentCount: 0,
                totalExpected: 0,
                attendancePct: 0,
              ),
      );
      _dailyCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return null;
    }
  }

  Future<List<ClassRosterAttendance>> classRoster({
    required String classId,
    DateTime? date,
    bool force = false,
  }) async {
    date ??= DateTime.now();
    final dateStr = DateFormat('yyyy-MM-dd').format(date);
    final key = '$classId:$dateStr';
    if (!force && _classRosterCache.containsKey(key)) {
      return _classRosterCache[key]!;
    }
    try {
      _setLoading(true);
      final res = await _api.get<List<ClassRosterAttendance>>(
        '/attendance/class/$classId/roster/',
        query: {'date': dateStr},
        decode: (d) {
          final list = (d is Map ? (d['data'] as List?) : d as List?) ?? [];
          return list
              .map((e) => ClassRosterAttendance.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
              .toList();
        },
      );
      _classRosterCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    } finally {
      _setLoading(false);
    }
  }

  Future<StudentAttendanceRange?> studentRange({
    required String studentId,
    DateTime? start,
    DateTime? end,
    bool force = false,
  }) async {
    start ??= DateTime.now().subtract(const Duration(days: 30));
    end ??= DateTime.now();
    final s = DateFormat('yyyy-MM-dd').format(start);
    final e = DateFormat('yyyy-MM-dd').format(end);
    final key = '$studentId:$s:$e';
    if (!force && _studentRangeCache.containsKey(key)) {
      return _studentRangeCache[key];
    }
    try {
      _setLoading(true);
      final res = await _api.get<StudentAttendanceRange>(
        '/attendance/students/$studentId/range/',
        query: {'start_date': s, 'end_date': e},
        decode: (d) => d is Map
            ? StudentAttendanceRange.fromJson(d as Map<String, dynamic>)
            : StudentAttendanceRange(
                events: [],
                presentDays: 0,
                lateDays: 0,
                absentDays: 0,
                expectedDays: 0,
                attendancePct: 0,
              ),
      );
      _studentRangeCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return null;
    } finally {
      _setLoading(false);
    }
  }

  Future<StaffAttendanceRange?> myStaffAttendance({
    DateTime? start,
    DateTime? end,
    bool force = false,
  }) async {
    start ??= DateTime.now().subtract(const Duration(days: 30));
    end ??= DateTime.now();
    final s = DateFormat('yyyy-MM-dd').format(start);
    final e = DateFormat('yyyy-MM-dd').format(end);
    final key = '$s:$e';
    if (!force && _staffRangeCache.containsKey(key)) {
      return _staffRangeCache[key];
    }
    try {
      _setLoading(true);
      final res = await _api.get<StaffAttendanceRange>(
        '/attendance/staff/me/range/',
        query: {'start_date': s, 'end_date': e},
        decode: (d) => d is Map
            ? StaffAttendanceRange.fromJson(d as Map<String, dynamic>)
            : StaffAttendanceRange(
                events: [],
                presentDays: 0,
                lateDays: 0,
                absentDays: 0,
                expectedDays: 0,
                attendancePct: 0,
              ),
      );
      _staffRangeCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return null;
    } finally {
      _setLoading(false);
    }
  }

  Future<List<Student>> myWards({bool force = false}) async {
    const key = 'wards';
    if (!force && _wardCache.containsKey(key)) return _wardCache[key]!;
    try {
      _setLoading(true);
      final res = await _api.get<List<Student>>(
        '/students/',
        query: {'my_wards': 'true', 'page_size': '200'},
        decode: (d) {
          final list = (d is Map ? (d['data'] as List?) : d as List?) ?? [];
          return list
              .map((e) => Student.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
              .toList();
        },
      );
      _wardCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    } finally {
      _setLoading(false);
    }
  }

  Future<List<Announcement>> announcements({bool force = false}) async {
    const key = 'announcements';
    if (!force && _announcementsCache.containsKey(key)) {
      return _announcementsCache[key]!;
    }
    try {
      _setLoading(true);
      final res = await _api.get<List<Announcement>>(
        '/announcements/',
        query: {'page_size': '50'},
        decode: (d) {
          final list = (d is Map ? (d['data'] as List?) : d as List?) ?? [];
          return list
              .map((e) => Announcement.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
              .toList();
        },
      );
      _announcementsCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    } finally {
      _setLoading(false);
    }
  }

  Future<List<InvoiceSummary>> myInvoices({String? studentId, bool force = false}) async {
    final key = 'invoices:${studentId ?? 'all'}';
    if (!force && _invoiceCache.containsKey(key)) return _invoiceCache[key]!;
    try {
      _setLoading(true);
      final res = await _api.get<List<InvoiceSummary>>(
        '/invoices/',
        query: {if (studentId != null) 'student_id': studentId, 'page_size': '50'},
        decode: (d) {
          final list = (d is Map ? (d['data'] as List?) : d as List?) ?? [];
          return list
              .map((e) => InvoiceSummary.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
              .toList();
        },
      );
      _invoiceCache[key] = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    } finally {
      _setLoading(false);
    }
  }

  Future<List<ClassGroup>> fetchMyClasses({bool force = false}) async {
    if (!force && _myClasses.isNotEmpty) return _myClasses;
    try {
      _setLoading(true);
      final res = await _api.get<List<ClassGroup>>(
        '/classes/',
        query: {'my_classes': 'true', 'page_size': '100'},
        decode: (d) {
          final list = (d is Map ? (d['data'] as List?) : d as List?) ?? [];
          return list
              .map((e) => ClassGroup.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
              .toList();
        },
      );
      _myClasses = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    } finally {
      _setLoading(false);
    }
  }

  void clearCache() {
    _dailyCache.clear();
    _studentRangeCache.clear();
    _staffRangeCache.clear();
    _classRosterCache.clear();
    _wardCache.clear();
    _announcementsCache.clear();
    _invoiceCache.clear();
    _myClasses = [];
    notifyListeners();
  }
}
