import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/api_client.dart';
import '../constants/app_constants.dart';
import '../models/user.dart';

class AuthProvider extends ChangeNotifier {
  final ApiClient api = ApiClient();
  final FlutterSecureStorage _secure = const FlutterSecureStorage();
  SharedPreferences? _prefs;

  AppUser? _user;
  String? _accessToken;
  bool _isLoading = true;
  String? _error;

  AppUser? get user => _user;
  String? get accessToken => _accessToken;
  bool get isLoading => _isLoading;
  bool get isAuthenticated => _user != null && _accessToken != null;
  String? get error => _error;
  UserRole? get role => _user?.role;
  bool get isTeacher => _user?.role.isTeacher ?? false;
  bool get isParent => _user?.role == UserRole.parent;
  bool get isStudent => _user?.role == UserRole.student;
  bool get isStaff => _user?.role.isStaff ?? false;

  Future<void> init() async {
    _prefs = await SharedPreferences.getInstance();
    try {
      final token = await _secure.read(key: StorageKeys.accessToken);
      final userStr = _prefs!.getString(StorageKeys.userData);
      if (token != null && userStr != null) {
        final userJson = jsonDecode(userStr);
        _user = AppUser.fromJson(userJson is Map<String, dynamic> ? userJson : <String, dynamic>{});
        _accessToken = token;
        api.updateToken(token);
      }
    } catch (_) {}
    _isLoading = false;
    notifyListeners();
  }

  Future<bool> login({required String email, required String password}) async {
    _error = null;
    _isLoading = true;
    notifyListeners();
    try {
      final resp = await api.post<LoginResponse>(
        '/auth/login/',
        body: {'email': email, 'password': password},
        decode: (data) {
          if (data is Map) return LoginResponse.fromJson(data as Map<String, dynamic>);
          throw 'invalid login response';
        },
      );
      _user = resp.user;
      _accessToken = resp.accessToken;
      api.updateToken(resp.accessToken);
      await _secure.write(key: StorageKeys.accessToken, value: resp.accessToken);
      if (resp.refreshToken != null) {
        await _secure.write(key: StorageKeys.refreshToken, value: resp.refreshToken!);
      }
      await _prefs!.setString(StorageKeys.userData, jsonEncode(_user!.toJson()));
      await _prefs!.setString(StorageKeys.userRole, _user!.role.rawName);
      _isLoading = false;
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      _error = e.message;
    } catch (e) {
      _error = 'Login failed. Please try again.';
    }
    _isLoading = false;
    notifyListeners();
    return false;
  }

  Future<void> logout() async {
    _user = null;
    _accessToken = null;
    api.updateToken(null);
    await _secure.delete(key: StorageKeys.accessToken);
    await _secure.delete(key: StorageKeys.refreshToken);
    await _prefs?.remove(StorageKeys.userData);
    await _prefs?.remove(StorageKeys.userRole);
    notifyListeners();
  }
}
