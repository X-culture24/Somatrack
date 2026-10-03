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

  /// For binary responses (PDF invoices/receipts) — the JSON-decoding get<T>
  /// above can't be reused since there's no JSON body to parse.
  Future<List<int>> getBytes(String path) async {
    final uri = _uri(path);
    final response = await _client
        .get(uri, headers: _headers(json: false))
        .timeout(ApiConstants.timeout);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      String message = 'Request failed: ${response.statusCode}';
      try {
        final body = jsonDecode(utf8.decode(response.bodyBytes));
        if (body is Map && body.containsKey('message')) message = body['message'] ?? message;
      } catch (_) {}
      throw ApiException(response.statusCode, message);
    }
    return response.bodyBytes;
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

/// Direct-to-DB gateway resources (students, classes, lesson-plans, announcements,
/// teacher/my-classes...) wrap list responses as {"results": [...], "count": N};
/// finance-svc-backed resources (invoices, payments) use {"data": [...]} instead.
/// Same shape, different key — mirrors the web app's api/client.ts unwrap().
/// Several screens here previously checked only 'data', so a real 'results'
/// response (the more common shape) silently decoded to an empty list.
List<dynamic> unwrapList(dynamic d) {
  if (d is List) return d;
  if (d is Map) {
    if (d['results'] is List) return d['results'] as List;
    if (d['data'] is List) return d['data'] as List;
  }
  return const [];
}

class ApiException implements Exception {
  final int statusCode;
  final String message;
  ApiException(this.statusCode, this.message);

  @override
  String toString() => 'ApiException($statusCode): $message';
}
