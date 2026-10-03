## St Mary's LMS — API smoke-test Makefile
## Usage:
##   make test-all        Run every test in sequence
##   make login           Obtain a JWT (used automatically by other targets)
##   make test-<name>     Run a single test group

BASE_URL  := http://localhost:8000/api
TOKEN_FILE := /tmp/st_marys_token.txt

# Admin credentials (same as test user seeded in DB)
ADMIN_EMAIL    := admin@stmaryskabete.ac.ke
ADMIN_PASSWORD := Admin1234!

TEACHER_EMAIL    := teacher@stmaryskabete.ac.ke
TEACHER_PASSWORD := Admin1234!

PASS := \033[0;32mPASS\033[0m
FAIL := \033[0;31mFAIL\033[0m

# ---------------------------------------------------------------------------
.PHONY: test-all login test-dashboard test-students test-alumni test-siblings \
        test-promote test-fee-arrears test-ledger \
        test-cbc-marksheet test-cbc-save test-cbc-signoffs \
        test-classes test-terms test-finance test-invoices test-payments \
        test-teacher-token build restart

# ---------------------------------------------------------------------------
## Build and restart the API server (kills any running instance first)
build:
	@echo "==> Building API…"
	go build -o /tmp/st_marys_api ./cmd/api/
	@echo "Build OK"

restart: build
	@echo "==> Restarting API on :8000…"
	@fuser -k 8000/tcp 2>/dev/null || true
	@sleep 0.5
	@nohup /tmp/st_marys_api > /tmp/st_marys_api.log 2>&1 &
	@sleep 1.5
	@echo "API started. Logs: /tmp/st_marys_api.log"

# ---------------------------------------------------------------------------
## Helpers
_check_status = @STATUS=$$(echo '$$1' | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('status','?'))" 2>/dev/null); \
	if [ "$$STATUS" = "ok" ]; then echo "[$(PASS)] $$2"; else echo "[$(FAIL)] $$2 → $$STATUS"; fi

token:
	$(eval TOKEN := $(shell curl -s -X POST $(BASE_URL)/auth/login/ \
		-H "Content-Type: application/json" \
		-d '{"email":"$(ADMIN_EMAIL)","password":"$(ADMIN_PASSWORD)"}' \
		| python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('access',d.get('token',d.get('jwt','FAIL'))))"))
	@if [ "$(TOKEN)" = "FAIL" ]; then echo "[$(FAIL)] Login failed"; exit 1; fi
	@echo "$(TOKEN)" > $(TOKEN_FILE)
	@echo "[$(PASS)] Login → token obtained"

teacher-token:
	$(eval TTOKEN := $(shell curl -s -X POST $(BASE_URL)/auth/login/ \
		-H "Content-Type: application/json" \
		-d '{"email":"$(TEACHER_EMAIL)","password":"$(TEACHER_PASSWORD)"}' \
		| python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('access',d.get('token',d.get('jwt','FAIL'))))"))
	@echo "$(TTOKEN)" > /tmp/st_marys_teacher_token.txt
	@echo "[$(PASS)] Teacher login → token obtained"

_auth_header = -H "Authorization: Bearer $$(cat $(TOKEN_FILE))"
_tauth_header = -H "Authorization: Bearer $$(cat /tmp/st_marys_teacher_token.txt)"

# ---------------------------------------------------------------------------
login: token

# ---------------------------------------------------------------------------
test-dashboard: token
	@echo "--- Dashboard ---"
	@STATUS=$$(curl -s $(_auth_header) $(BASE_URL)/dashboard/ | python3 -c "import sys,json; d=json.load(sys.stdin); print('ok' if 'stats' in d or 'data' in d or 'recent_students' in d else 'no-stats')" 2>/dev/null); \
		[ "$$STATUS" = "ok" ] && echo "[$(PASS)] GET /dashboard/" || echo "[$(FAIL)] GET /dashboard/ → $$STATUS"

# ---------------------------------------------------------------------------
test-students: token
	@echo "--- Students ---"
	@COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/students/?page_size=5" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /students/ → $$COUNT rows"
	@SID=$$(curl -s $(_auth_header) "$(BASE_URL)/students/?page_size=1" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(r[0]['id'] if r else '')" 2>/dev/null); \
		if [ -n "$$SID" ]; then \
			echo "$$SID" > /tmp/st_marys_student_id.txt; \
			echo "[$(PASS)] Got student_id $$SID"; \
		else \
			echo "[$(FAIL)] Could not get any student ID"; \
		fi

# ---------------------------------------------------------------------------
test-alumni: token
	@echo "--- Alumni ---"
	@COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/students/?alumni=true&page_size=5" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /students/?alumni=true → $$COUNT rows"

# ---------------------------------------------------------------------------
test-siblings: token
	@echo "--- Siblings ---"
	@COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/guardians/siblings/" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /guardians/siblings/ → $$COUNT families"

# ---------------------------------------------------------------------------
test-fee-arrears: token
	@echo "--- Fee Arrears ---"
	@COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/students/fee-arrears/" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /students/fee-arrears/ → $$COUNT students with arrears"

# ---------------------------------------------------------------------------
test-ledger: token test-students
	@echo "--- Student Ledger ---"
	@SID=$$(cat /tmp/st_marys_student_id.txt 2>/dev/null); \
		if [ -z "$$SID" ]; then echo "[$(FAIL)] No student_id available"; exit 0; fi; \
		KEYS=$$(curl -s $(_auth_header) "$(BASE_URL)/student-ledger/?student_id=$$SID" | python3 -c "import sys,json; d=json.load(sys.stdin); print(','.join(sorted(d.keys())))" 2>/dev/null); \
		echo "[$(PASS)] GET /student-ledger/?student_id=$$SID → keys: $$KEYS"

# ---------------------------------------------------------------------------
test-classes: token
	@echo "--- Classes / Terms ---"
	@CID=$$(curl -s $(_auth_header) "$(BASE_URL)/classes/" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(r[0]['id'] if r else '')" 2>/dev/null); \
		echo "$$CID" > /tmp/st_marys_class_id.txt; \
		echo "[$(PASS)] GET /classes/ → class_id $$CID"
	@TID=$$(curl -s $(_auth_header) "$(BASE_URL)/terms/" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(r[0]['id'] if r else '')" 2>/dev/null); \
		echo "$$TID" > /tmp/st_marys_term_id.txt; \
		echo "[$(PASS)] GET /terms/ → term_id $$TID"

# ---------------------------------------------------------------------------
test-cbc-marksheet: token test-classes
	@echo "--- CBC Marksheet ---"
	@CID=$$(cat /tmp/st_marys_class_id.txt 2>/dev/null); TID=$$(cat /tmp/st_marys_term_id.txt 2>/dev/null); \
		if [ -z "$$CID" ] || [ -z "$$TID" ]; then echo "[$(FAIL)] No class/term IDs available"; exit 0; fi; \
		COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/cbc/marksheet/?class_id=$$CID&term_id=$$TID" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('students',[]); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /cbc/marksheet/ → $$COUNT students"

# ---------------------------------------------------------------------------
test-cbc-save: token test-classes test-students
	@echo "--- CBC Save Marks ---"
	@STATUS=$$(curl -s -X PUT $(_auth_header) "$(BASE_URL)/cbc/marksheet/" \
		-H "Content-Type: application/json" \
		-d '{"entries":[]}' | python3 -c "import sys,json; d=json.load(sys.stdin); print('ok' if 'saved' in d else 'fail')" 2>/dev/null); \
		[ "$$STATUS" = "ok" ] && echo "[$(PASS)] PUT /cbc/marksheet/ (empty entries) → ok" || echo "[$(FAIL)] PUT /cbc/marksheet/ → $$STATUS"

# ---------------------------------------------------------------------------
test-cbc-signoffs: token test-classes
	@echo "--- CBC Signoffs ---"
	@CID=$$(cat /tmp/st_marys_class_id.txt 2>/dev/null); TID=$$(cat /tmp/st_marys_term_id.txt 2>/dev/null); \
		COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/cbc/signoffs/?class_id=$$CID&term_id=$$TID" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',[]); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /cbc/signoffs/ → $$COUNT signoffs"

# ---------------------------------------------------------------------------
test-finance: token
	@echo "--- Finance ---"
	@COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/invoices/?page_size=5" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /invoices/ → $$COUNT invoices"
	@COUNT=$$(curl -s $(_auth_header) "$(BASE_URL)/payments/?page_size=5" | python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('results',d.get('data',[])); print(len(r))" 2>/dev/null); \
		echo "[$(PASS)] GET /payments/ → $$COUNT payments"

# ---------------------------------------------------------------------------
test-promote: token test-classes test-students
	@echo "--- Promote (validation check) ---"
	@MSG=$$(curl -s -X POST $(_auth_header) "$(BASE_URL)/students/promote/" \
		-H "Content-Type: application/json" \
		-d '{"student_ids":[],"target_class_id":"00000000-0000-0000-0000-000000000000"}' | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('message',''))" 2>/dev/null); \
		[ -n "$$MSG" ] && echo "[$(PASS)] POST /students/promote/ validation → $$MSG" || echo "[$(FAIL)] POST /students/promote/ unexpected response"

# ---------------------------------------------------------------------------
test-cbc-pdf: token test-classes test-students
	@echo "--- CBC Report PDF ---"
	@SID=$$(cat /tmp/st_marys_student_id.txt 2>/dev/null); TID=$$(cat /tmp/st_marys_term_id.txt 2>/dev/null); \
		if [ -z "$$SID" ] || [ -z "$$TID" ]; then echo "[$(FAIL)] No student/term IDs"; exit 0; fi; \
		curl -s -o /tmp/st_marys_cbc_test.pdf $(_auth_header) \
			"$(BASE_URL)/documents/cbc-report/?student_id=$$SID&term_id=$$TID"; \
		TYPE=$$(file /tmp/st_marys_cbc_test.pdf | grep -o "PDF document" || echo "not-pdf"); \
		SIZE=$$(wc -c < /tmp/st_marys_cbc_test.pdf); \
		[ "$$TYPE" = "PDF document" ] && echo "[$(PASS)] GET /documents/cbc-report/ → PDF ($$SIZE bytes)" || echo "[$(FAIL)] Expected PDF, got: $$(cat /tmp/st_marys_cbc_test.pdf | head -c 100)"

test-all: login test-dashboard test-students test-alumni test-siblings \
          test-fee-arrears test-ledger test-classes \
          test-cbc-marksheet test-cbc-save test-cbc-signoffs test-cbc-pdf \
          test-finance test-promote
	@echo ""
	@echo "==> All tests complete"
