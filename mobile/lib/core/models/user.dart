enum UserRole {
  systemAdmin,
  schoolDirector,
  financeOfficer,
  headTeacher,
  deputyHeadTeacher,
  academicCoordinator,
  classTeacher,
  subjectTeacher,
  librarian,
  transportCoordinator,
  admissionsOfficer,
  receptionist,
  parent,
  student;

  static UserRole fromString(String? s) {
    if (s == null) return UserRole.parent;
    switch (s.toUpperCase()) {
      case 'SYSTEM_ADMIN': return UserRole.systemAdmin;
      case 'SCHOOL_DIRECTOR': return UserRole.schoolDirector;
      case 'FINANCE_OFFICER': return UserRole.financeOfficer;
      case 'HEAD_TEACHER': return UserRole.headTeacher;
      case 'DEPUTY_HEAD_TEACHER': return UserRole.deputyHeadTeacher;
      case 'ACADEMIC_COORDINATOR': return UserRole.academicCoordinator;
      case 'CLASS_TEACHER': return UserRole.classTeacher;
      case 'SUBJECT_TEACHER': return UserRole.subjectTeacher;
      case 'LIBRARIAN': return UserRole.librarian;
      case 'TRANSPORT_COORDINATOR': return UserRole.transportCoordinator;
      case 'ADMISSIONS_OFFICER': return UserRole.admissionsOfficer;
      case 'RECEPTIONIST': return UserRole.receptionist;
      case 'PARENT': return UserRole.parent;
      case 'STUDENT': return UserRole.student;
      default: return UserRole.parent;
    }
  }

  String get rawName => name;

  String get displayName {
    switch (this) {
      case UserRole.systemAdmin: return 'System Admin';
      case UserRole.schoolDirector: return 'School Director';
      case UserRole.financeOfficer: return 'Finance Officer';
      case UserRole.headTeacher: return 'Head Teacher';
      case UserRole.deputyHeadTeacher: return 'Deputy Head';
      case UserRole.academicCoordinator: return 'Academic Coordinator';
      case UserRole.classTeacher: return 'Class Teacher';
      case UserRole.subjectTeacher: return 'Subject Teacher';
      case UserRole.librarian: return 'Librarian';
      case UserRole.transportCoordinator: return 'Transport Coordinator';
      case UserRole.admissionsOfficer: return 'Admissions Officer';
      case UserRole.receptionist: return 'Receptionist';
      case UserRole.parent: return 'Parent';
      case UserRole.student: return 'Student';
    }
  }

  bool get isStaff {
    return switch (this) {
      UserRole.parent || UserRole.student => false,
      _ => true,
    };
  }

  bool get isTeacher {
    return switch (this) {
      UserRole.classTeacher ||
      UserRole.subjectTeacher ||
      UserRole.headTeacher ||
      UserRole.deputyHeadTeacher ||
      UserRole.academicCoordinator => true,
      _ => false,
    };
  }
}

class AppUser {
  final String id;
  final String email;
  final String firstName;
  final String lastName;
  final UserRole role;
  final String? phoneNumber;
  final bool isActive;

  AppUser({
    required this.id,
    required this.email,
    required this.firstName,
    required this.lastName,
    required this.role,
    this.phoneNumber,
    this.isActive = true,
  });

  factory AppUser.fromJson(Map<String, dynamic> json) {
    return AppUser(
      id: json['id']?.toString() ?? '',
      email: json['email'] ?? '',
      firstName: json['first_name'] ?? '',
      lastName: json['last_name'] ?? '',
      role: UserRole.fromString(json['role']),
      phoneNumber: json['phone_number'],
      isActive: json['is_active'] ?? true,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'email': email,
        'first_name': firstName,
        'last_name': lastName,
        'role': role.rawName.toUpperCase(),
        'phone_number': phoneNumber,
        'is_active': isActive,
      };

  String get fullName => '$firstName $lastName';

  String get initials =>
      '${firstName.isNotEmpty ? firstName[0] : ''}${lastName.isNotEmpty ? lastName[0] : ''}'
          .toUpperCase();
}

class LoginResponse {
  final String accessToken;
  final String? refreshToken;
  final AppUser user;

  LoginResponse({
    required this.accessToken,
    this.refreshToken,
    required this.user,
  });

  factory LoginResponse.fromJson(Map<String, dynamic> json) {
    final userJson = json['user'] ?? json;
    return LoginResponse(
      accessToken: json['access'] ?? json['access_token'] ?? json['token'] ?? '',
      refreshToken: json['refresh'] ?? json['refresh_token'],
      user: AppUser.fromJson(userJson is Map<String, dynamic> ? userJson : <String, dynamic>{}),
    );
  }
}
