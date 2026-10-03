class AttendanceStatus {
  static const String present = 'present';
  static const String late = 'late';
  static const String absent = 'absent';
}

class DailyAttendanceSummary {
  final String date;
  final int presentCount;
  final int lateCount;
  final int absentCount;
  final int totalExpected;
  final double attendancePct;

  DailyAttendanceSummary({
    required this.date,
    required this.presentCount,
    required this.lateCount,
    required this.absentCount,
    required this.totalExpected,
    required this.attendancePct,
  });

  factory DailyAttendanceSummary.fromJson(Map<String, dynamic> json) {
    return DailyAttendanceSummary(
      date: json['date'] ?? '',
      presentCount: (json['present_count'] ?? 0).toInt(),
      lateCount: (json['late_count'] ?? 0).toInt(),
      absentCount: (json['absent_count'] ?? 0).toInt(),
      totalExpected: (json['total_expected'] ?? 0).toInt(),
      attendancePct: (json['attendance_pct'] ?? 0.0).toDouble(),
    );
  }
}

class StudentAttendanceEvent {
  final String date;
  final String status;
  final String? timeIn;
  final String? timeOut;
  final String remarks;
  final String? subject;

  StudentAttendanceEvent({
    required this.date,
    required this.status,
    this.timeIn,
    this.timeOut,
    this.remarks = '',
    this.subject,
  });

  factory StudentAttendanceEvent.fromJson(Map<String, dynamic> json) {
    return StudentAttendanceEvent(
      date: json['date'] ?? '',
      status: json['status'] ?? '',
      timeIn: json['time_in'],
      timeOut: json['time_out'],
      remarks: json['remarks'] ?? '',
      subject: json['subject'],
    );
  }

  bool get isPresent => status == AttendanceStatus.present;
  bool get isLate => status == AttendanceStatus.late;
  bool get isAbsent => status == AttendanceStatus.absent;
}

class StudentAttendanceRange {
  final List<StudentAttendanceEvent> events;
  final int presentDays;
  final int lateDays;
  final int absentDays;
  final int expectedDays;
  final double attendancePct;

  StudentAttendanceRange({
    required this.events,
    required this.presentDays,
    required this.lateDays,
    required this.absentDays,
    required this.expectedDays,
    required this.attendancePct,
  });

  factory StudentAttendanceRange.fromJson(Map<String, dynamic> json) {
    final list = (json['events'] as List?) ?? [];
    return StudentAttendanceRange(
      events: list
          .map((e) => StudentAttendanceEvent.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
          .toList(),
      presentDays: (json['present_days'] ?? 0).toInt(),
      lateDays: (json['late_days'] ?? 0).toInt(),
      absentDays: (json['absent_days'] ?? 0).toInt(),
      expectedDays: (json['expected_days'] ?? 0).toInt(),
      attendancePct: (json['attendance_pct'] ?? 0.0).toDouble(),
    );
  }
}

class StaffAttendanceEvent {
  final String date;
  final String status;
  final String? timeIn;
  final String? timeOut;

  StaffAttendanceEvent({
    required this.date,
    required this.status,
    this.timeIn,
    this.timeOut,
  });

  factory StaffAttendanceEvent.fromJson(Map<String, dynamic> json) {
    return StaffAttendanceEvent(
      date: json['date'] ?? '',
      status: json['status'] ?? '',
      timeIn: json['time_in'],
      timeOut: json['time_out'],
    );
  }
}

class StaffAttendanceRange {
  final List<StaffAttendanceEvent> events;
  final int presentDays;
  final int lateDays;
  final int absentDays;
  final int expectedDays;
  final double attendancePct;

  StaffAttendanceRange({
    required this.events,
    required this.presentDays,
    required this.lateDays,
    required this.absentDays,
    required this.expectedDays,
    required this.attendancePct,
  });

  factory StaffAttendanceRange.fromJson(Map<String, dynamic> json) {
    final list = (json['events'] as List?) ?? [];
    return StaffAttendanceRange(
      events: list
          .map((e) => StaffAttendanceEvent.fromJson(e is Map<String, dynamic> ? e : <String, dynamic>{}))
          .toList(),
      presentDays: (json['present_days'] ?? 0).toInt(),
      lateDays: (json['late_days'] ?? 0).toInt(),
      absentDays: (json['absent_days'] ?? 0).toInt(),
      expectedDays: (json['expected_days'] ?? 0).toInt(),
      attendancePct: (json['attendance_pct'] ?? 0.0).toDouble(),
    );
  }
}

class ClassRosterAttendance {
  final String studentId;
  final String studentName;
  final String admissionNo;
  final String? classId;
  final String className;
  final int presentDays;
  final int lateDays;
  final int absentDays;
  final int expectedDays;
  final double attendancePct;

  ClassRosterAttendance({
    required this.studentId,
    required this.studentName,
    required this.admissionNo,
    this.classId,
    required this.className,
    required this.presentDays,
    required this.lateDays,
    required this.absentDays,
    required this.expectedDays,
    required this.attendancePct,
  });

  factory ClassRosterAttendance.fromJson(Map<String, dynamic> json) {
    return ClassRosterAttendance(
      studentId: json['student_id']?.toString() ?? '',
      studentName: json['student_name'] ?? '',
      admissionNo: json['admission_no'] ?? '',
      classId: json['class_id']?.toString(),
      className: json['class_name'] ?? '',
      presentDays: (json['present_days'] ?? 0).toInt(),
      lateDays: (json['late_days'] ?? 0).toInt(),
      absentDays: (json['absent_days'] ?? 0).toInt(),
      expectedDays: (json['expected_days'] ?? 0).toInt(),
      attendancePct: (json['attendance_pct'] ?? 0.0).toDouble(),
    );
  }
}

class Student {
  final String id;
  final String admissionNo;
  final String fullName;
  final String? firstName;
  final String? lastName;
  final String? className;
  final String? classId;
  final String status;

  Student({
    required this.id,
    required this.admissionNo,
    required this.fullName,
    this.firstName,
    this.lastName,
    this.className,
    this.classId,
    this.status = 'active',
  });

  factory Student.fromJson(Map<String, dynamic> json) {
    final first = json['first_name'] ?? '';
    final last = json['last_name'] ?? '';
    final full = json['full_name'] ?? json['search_name'] ?? '$first $last';
    return Student(
      id: json['id']?.toString() ?? '',
      admissionNo: json['admission_no'] ?? '',
      fullName: full.toString().trim(),
      firstName: first,
      lastName: last,
      className: json['class_name'],
      classId: json['current_class_id']?.toString() ?? json['class_id']?.toString(),
      status: json['status'] ?? 'active',
    );
  }

  String get initials {
    if (firstName != null && lastName != null) {
      return '${firstName!.isNotEmpty ? firstName![0] : ''}${lastName!.isNotEmpty ? lastName![0] : ''}'
          .toUpperCase();
    }
    if (fullName.isEmpty) return '';
    final parts = fullName.split(' ');
    return '${parts[0][0]}${parts.length > 1 ? parts.last[0] : ''}'.toUpperCase();
  }
}

class ClassGroup {
  final String id;
  final String grade;
  final String streamName;
  final String? academicYearId;
  final String? displayName;
  final int studentCount;
  final bool isHomeroom;
  final List<String> subjectsTaught;

  ClassGroup({
    required this.id,
    required this.grade,
    required this.streamName,
    this.academicYearId,
    this.displayName,
    this.studentCount = 0,
    this.isHomeroom = false,
    this.subjectsTaught = const [],
  });

  factory ClassGroup.fromJson(Map<String, dynamic> json) {
    return ClassGroup(
      id: json['id']?.toString() ?? '',
      // grade is a real integer on the backend (Playgroup=-2..Grade 13) —
      // toString() it rather than assigning the raw dynamic value, which
      // threw a "type 'int' is not a subtype of type 'String'" at runtime.
      grade: json['grade']?.toString() ?? '',
      streamName: json['stream_name'] ?? json['stream'] ?? '',
      academicYearId: json['academic_year_id']?.toString(),
      displayName: json['name']?.toString(),
      studentCount: (json['student_count'] ?? 0) is int
          ? json['student_count'] as int
          : int.tryParse(json['student_count'].toString()) ?? 0,
      isHomeroom: json['is_homeroom'] == true,
      subjectsTaught: ((json['subjects_taught'] as List?) ?? [])
          .map((e) => e.toString())
          .toList(),
    );
  }

  // /teacher/my-classes/ already sends a real display name ("Grade 4 West");
  // fall back to grade-stream only for callers still on the old /classes/ shape.
  String get name => (displayName != null && displayName!.isNotEmpty) ? displayName! : '$grade-$streamName';
}

class LessonPlan {
  final String id;
  final String classId;
  final String className;
  final String subjectId;
  final String subjectName;
  final int weekNumber;
  final String lessonDate;
  final String topic;
  final String subTopic;
  final String learningObjectives;
  final String status;

  LessonPlan({
    required this.id,
    required this.classId,
    required this.className,
    required this.subjectId,
    required this.subjectName,
    required this.weekNumber,
    required this.lessonDate,
    required this.topic,
    this.subTopic = '',
    this.learningObjectives = '',
    this.status = 'draft',
  });

  factory LessonPlan.fromJson(Map<String, dynamic> json) {
    return LessonPlan(
      id: json['id']?.toString() ?? '',
      classId: json['class_id']?.toString() ?? '',
      className: json['class_name'] ?? '',
      subjectId: json['subject_id']?.toString() ?? '',
      subjectName: json['subject_name'] ?? '',
      weekNumber: (json['week_number'] ?? 0) is int
          ? json['week_number'] as int
          : int.tryParse(json['week_number'].toString()) ?? 0,
      lessonDate: json['lesson_date'] ?? '',
      topic: json['topic'] ?? '',
      subTopic: json['sub_topic'] ?? '',
      learningObjectives: json['learning_objectives'] ?? '',
      status: json['status'] ?? 'draft',
    );
  }
}

class Subject {
  final String id;
  final String name;

  Subject({required this.id, required this.name});

  factory Subject.fromJson(Map<String, dynamic> json) {
    return Subject(id: json['id']?.toString() ?? '', name: json['name'] ?? '');
  }
}

class Announcement {
  final String id;
  final String title;
  final String body;
  final String priority;
  final DateTime? createdAt;
  final String audience;

  Announcement({
    required this.id,
    required this.title,
    required this.body,
    required this.priority,
    this.createdAt,
    required this.audience,
  });

  factory Announcement.fromJson(Map<String, dynamic> json) {
    return Announcement(
      id: json['id']?.toString() ?? '',
      title: json['title'] ?? '',
      body: json['body'] ?? json['content'] ?? '',
      priority: json['priority'] ?? 'normal',
      // Real column is publish_date — created_at kept as a fallback in case
      // a future endpoint adds one, but publish_date is what's actually sent.
      createdAt: DateTime.tryParse((json['publish_date'] ?? json['created_at'] ?? '').toString()),
      audience: json['audience'] ?? 'all',
    );
  }
}

class InvoiceSummary {
  final String id;
  final String studentId;
  final double totalDue;
  final double totalPaid;
  final double balance;
  final String status;
  final String? termName;

  InvoiceSummary({
    required this.id,
    required this.studentId,
    required this.totalDue,
    required this.totalPaid,
    required this.balance,
    required this.status,
    this.termName,
  });

  factory InvoiceSummary.fromJson(Map<String, dynamic> json) {
    return InvoiceSummary(
      id: json['id']?.toString() ?? '',
      studentId: json['student_id']?.toString() ?? '',
      totalDue: (json['total_due'] ?? 0).toDouble(),
      totalPaid: (json['total_paid'] ?? 0).toDouble(),
      balance: (json['balance'] ?? 0).toDouble(),
      status: json['status'] ?? 'unpaid',
      termName: json['term_name'],
    );
  }
}
