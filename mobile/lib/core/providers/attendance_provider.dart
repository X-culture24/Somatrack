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
  List<LessonPlan> _lessonPlans = [];
  List<Subject> _subjects = [];

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
          final list = unwrapList(d);
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
          final list = unwrapList(d);
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
          final list = unwrapList(d);
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
          final list = unwrapList(d);
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
        '/teacher/my-classes/',
        decode: (d) {
          final list = unwrapList(d);
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

  /// Raw PDF bytes for an invoice — internal/api/documents.go. Scoped
  /// server-side to the caller's own children, same as myInvoices above.
  Future<List<int>> invoicePdfBytes(String invoiceId) {
    return _api.getBytes('/invoices/$invoiceId/pdf/');
  }

  /// Triggers a real M-Pesa STK Push (internal/api/pay_fees.go) — a PIN
  /// prompt pops on `phone` and, once entered, the payment is applied to
  /// the student's invoice automatically. Returns the STK push request id
  /// to poll via [pollPaymentStatus].
  Future<String> initiatePayment({
    required String studentId,
    required String phone,
    required double amount,
  }) async {
    final id = await _api.post<String>(
      '/finance/pay/',
      body: {'student_id': studentId, 'phone': phone, 'amount': amount},
      decode: (d) => (d as Map)['id']?.toString() ?? '',
    );
    if (id.isEmpty) throw ApiException(0, 'No payment request id returned');
    return id;
  }

  /// Polls until the async callback lands (typically 5-20s) or times out.
  /// Returns the final status string: success | failed | cancelled | pending.
  Future<Map<String, dynamic>> pollPaymentStatus(String requestId, {int maxAttempts = 20}) async {
    for (var i = 0; i < maxAttempts; i++) {
      final res = await _api.get<Map<String, dynamic>>(
        '/finance/pay/$requestId/status/',
        decode: (d) => d is Map ? d.cast<String, dynamic>() : <String, dynamic>{},
      );
      final status = res['status'] as String? ?? 'pending';
      if (status == 'success' || status == 'failed' || status == 'cancelled') {
        return res;
      }
      await Future.delayed(const Duration(seconds: 3));
    }
    return {'status': 'pending', 'result_desc': 'Still waiting on M-Pesa — check back shortly.'};
  }

  /// internal/api/attendance_mark.go — the counterpart the "Take Attendance"
  /// screen was missing entirely; it used to fake a 1s delay and a success
  /// message without sending anything. Throws ApiException on failure.
  Future<void> markAttendance({
    required String classId,
    required String date,
    required Map<String, String> statusByStudentId,
  }) async {
    await _api.post<void>(
      '/attendance/mark/',
      body: {
        'class_id': classId,
        'date': date,
        'entries': statusByStudentId.entries
            .map((e) => {'student_id': e.key, 'status': e.value})
            .toList(),
      },
      decode: (_) {},
    );
    _classRosterCache.remove('$classId:$date');
    _dailyCache.clear();
  }

  List<LessonPlan> get lessonPlans => _lessonPlans;
  List<Subject> get subjects => _subjects;

  Future<List<LessonPlan>> fetchLessonPlans({bool force = false}) async {
    if (!force && _lessonPlans.isNotEmpty) return _lessonPlans;
    try {
      _setLoading(true);
      final res = await _api.get<List<LessonPlan>>(
        '/lesson-plans/',
        decode: (d) => unwrapList(d)
            .map((e) => LessonPlan.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
            .toList(),
      );
      _lessonPlans = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    } finally {
      _setLoading(false);
    }
  }

  Future<List<Subject>> fetchSubjects({bool force = false}) async {
    if (!force && _subjects.isNotEmpty) return _subjects;
    try {
      final res = await _api.get<List<Subject>>(
        '/subjects/',
        decode: (d) => unwrapList(d)
            .map((e) => Subject.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
            .toList(),
      );
      _subjects = res;
      return res;
    } on ApiException catch (e) {
      _error = e.message;
      return [];
    }
  }

  /// Throws ApiException on failure (e.g. server-side validation) — callers
  /// show the message directly rather than this returning a bool, since the
  /// message is specific (e.g. which field failed) and worth surfacing.
  Future<void> createLessonPlan({
    required String classId,
    required String subjectId,
    required int weekNumber,
    required String lessonDate,
    required String topic,
    String subTopic = '',
    String learningObjectives = '',
  }) async {
    await _api.post<void>(
      '/lesson-plans/',
      body: {
        'class_id': classId,
        'subject_id': subjectId,
        'week_number': weekNumber,
        'lesson_date': lessonDate,
        'topic': topic,
        'sub_topic': subTopic,
        'learning_objectives': learningObjectives,
      },
      decode: (_) {},
    );
    await fetchLessonPlans(force: true);
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
    _lessonPlans = [];
    _subjects = [];
    notifyListeners();
  }
}
