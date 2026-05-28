#!/usr/bin/env bash
# run_tests.sh — запуск одной итерации нагрузочного тестирования
#
# Использование:
#   ./run_tests.sh [iteration_label]
#
# Зависимости: vegeta, python3

set -euo pipefail

HOST="${HOST:-http://localhost:8082}"
RATE="${RATE:-200}"
DURATION="${DURATION:-30s}"
LABEL="${1:-run_$(date +%Y%m%d_%H%M%S)}"
RESULTS_DIR="pref_test/results/${LABEL}"

mkdir -p "${RESULTS_DIR}"

echo "============================================"
echo " Итерация: ${LABEL}"
echo " Host:     ${HOST}"
echo " Rate:     ${RATE} req/s  Duration: ${DURATION}"
echo "============================================"

# ─── Шаг 1: Проверяем наличие сессий ────────────────────────────────────────
if [ ! -f pref_test/test_sessions.json ]; then
  echo "[!] test_sessions.json не найден."
  echo "    Запустите сначала: python3 generate_data.py --host http://localhost:8081"
  exit 1
fi

# ─── Шаг 2: Генерируем targets для inbox ────────────────────────────────────
echo ""
echo "[1/3] Генерируем targets для GET /email/inbox ..."
python pref_test/make_targets.py \
  --host "${HOST}" \
  --mode inbox \
  --count 2000 \
  --out "${RESULTS_DIR}/targets_inbox.txt"

# ─── Шаг 3: Нагрузочный тест — чтение инбокса ───────────────────────────────
echo ""
echo "[2/3] Тест чтения (GET /email/inbox)  Rate=${RATE}  Duration=${DURATION} ..."

# Исправлено: убран флаг -inputs, используется перенаправление
vegeta attack \
  -targets="${RESULTS_DIR}/targets_inbox.txt" \
  -rate="${RATE}" \
  -duration="${DURATION}" \
  > "${RESULTS_DIR}/inbox_raw.bin"

vegeta report \
  -type=text \
  "${RESULTS_DIR}/inbox_raw.bin" \
  | tee "${RESULTS_DIR}/inbox_report.txt"

vegeta report \
  -type=json \
  "${RESULTS_DIR}/inbox_raw.bin" \
  > "${RESULTS_DIR}/inbox_report.json"

vegeta plot \
  -title="Inbox latency — ${LABEL}" \
  "${RESULTS_DIR}/inbox_raw.bin" \
  > "${RESULTS_DIR}/inbox_plot.html"

# ─── Шаг 4: Генерируем targets для GET /email/{id} ──────────────────────────
echo ""
echo "[3/3] Генерируем targets для GET /email/{id} ..."
python pref_test/make_targets.py \
  --host "${HOST}" \
  --mode by_id \
  --count 2000 \
  --out "${RESULTS_DIR}/targets_by_id.txt"

vegeta attack \
  -targets="${RESULTS_DIR}/targets_by_id.txt" \
  -rate="${RATE}" \
  -duration="${DURATION}" \
  > "${RESULTS_DIR}/by_id_raw.bin"

vegeta report \
  -type=text \
  "${RESULTS_DIR}/by_id_raw.bin" \
  | tee "${RESULTS_DIR}/by_id_report.txt"

vegeta report \
  -type=json \
  "${RESULTS_DIR}/by_id_raw.bin" \
  > "${RESULTS_DIR}/by_id_report.json"

vegeta plot \
  -title="GetEmailByID latency — ${LABEL}" \
  "${RESULTS_DIR}/by_id_raw.bin" \
  > "${RESULTS_DIR}/by_id_plot.html"

echo ""
echo "============================================"
echo " Результаты сохранены в: ${RESULTS_DIR}/"
echo "============================================"