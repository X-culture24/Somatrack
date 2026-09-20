class ApiConstants {
  static const String baseUrl = String.fromEnvironment('API_BASE_URL',
      defaultValue: 'http://10.0.2.2:8000');
  static const String apiBase = '$baseUrl/api';
  static const Duration timeout = Duration(seconds: 30);
}

class AppConstants {
  static const String appName = "ACK St. Mary's Kabete";
  static const String schoolSlogan = "Strive for Excellence";
  static const String nairobiTz = 'Africa/Nairobi';
}

class StorageKeys {
  static const String accessToken = 'access_token';
  static const String refreshToken = 'refresh_token';
  static const String userData = 'user_data';
  static const String userRole = 'user_role';
}
