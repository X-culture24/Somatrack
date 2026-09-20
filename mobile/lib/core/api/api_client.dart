import 'dart:convert';
import 'package:http/http.dart' as http;

import '../constants/app_constants.dart';

class ApiClient {
  final http.Client _client;
  String? _accessToken;

  ApiClient({http.Client? client}) : _client = client ?? http.Client();

  void updateToken(String? token) {
    _accessToken = token;
  }

  Map<String, String> _headers({bool json = true}) {
    final h = <String, String>{
      if (json) 'Content-Type': 'application/json',
      if (_accessToken != null) 'Authorization': 'Bearer $_accessToken',
    };
    return h;
  }

  Uri _uri(String path, [Map<String, String>? query]) {
    final base = Uri.parse('${ApiConstants.apiBase}$path');
    if (query == null || query.isEmpty) return base;
    return base.replace(queryParameters: query);
  }

  Future<T> _handleResponse<T>(
    http.Response response,
    T Function(dynamic) decode,
  ) async {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      if (response.body.isEmpty) return decode(null);
      try {
        final decoded = jsonDecode(response.body);
        return decode(decoded);
      } catch (_) {
        return decode(null);
      }
    }
    String message = 'Request failed: ${response.statusCode}';
    try {
      final body = jsonDecode(response.body);
      if (body is Map && body.containsKey('message')) {
        message = body['message'] ?? message;
      }
      if (body is Map && body.containsKey('error')) {
        message = '${body['error']}: $message';
      }
    } catch (_) {}
    throw ApiException(response.statusCode, message);
  }

  Future<T> get<T>(
    String path, {
    Map<String, String>? query,
    required T Function(dynamic) decode,
  }) async {
    final uri = _uri(path, query);
    final response = await _client
        .get(uri, headers: _headers())
        .timeout(ApiConstants.timeout);
    return _handleResponse(response, decode);
  }

  Future<T> post<T>(
    String path, {
    Object? body,
    Map<String, String>? query,
    required T Function(dynamic) decode,
  }) async {
    final uri = _uri(path, query);
    final response = await _client
        .post(uri, headers: _headers(), body: body != null ? jsonEncode(body) : null)
        .timeout(ApiConstants.timeout);
    return _handleResponse(response, decode);
  }

  Future<T> put<T>(
    String path, {
    Object? body,
    required T Function(dynamic) decode,
  }) async {
    final uri = _uri(path);
    final response = await _client
        .put(uri, headers: _headers(), body: body != null ? jsonEncode(body) : null)
        .timeout(ApiConstants.timeout);
    return _handleResponse(response, decode);
  }

  Future<T> patch<T>(
    String path, {
    Object? body,
    required T Function(dynamic) decode,
  }) async {
    final uri = _uri(path);
    final response = await _client
        .patch(uri, headers: _headers(), body: body != null ? jsonEncode(body) : null)
        .timeout(ApiConstants.timeout);
    return _handleResponse(response, decode);
  }

  Future<void> delete(String path) async {
    final uri = _uri(path);
    final response = await _client
        .delete(uri, headers: _headers())
        .timeout(ApiConstants.timeout);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(response.statusCode, 'Delete failed');
    }
  }
}

class ApiException implements Exception {
  final int statusCode;
  final String message;
  ApiException(this.statusCode, this.message);

  @override
  String toString() => 'ApiException($statusCode): $message';
}
