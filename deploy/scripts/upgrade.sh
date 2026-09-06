#!/bin/bash
# Family Budget Service - Upgrade Script
# Safely upgrades the application with automatic backup and rollback

set -euo pipefail
# -E: ловушка ERR наследуется функциями и подоболочками, иначе она не сработает
# внутри start_service/verify_health, ради которых и заведена.
set -E

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR}/backups}"
DATA_DIR="${DATA_DIR:-${INSTALL_DIR}/data}"
# Клон репозитория: публичного образа в GHCR нет (D-02), образ собирается на месте,
# поэтому «версия» — это git-ref в этом каталоге, а не тег образа.
SRC_DIR="${SRC_DIR:-${INSTALL_DIR}/src}"
COMPOSE_FILE="${COMPOSE_FILE:-${INSTALL_DIR}/docker-compose.yml}"
CADDYFILE="${CADDYFILE:-${INSTALL_DIR}/caddy/Caddyfile}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8080/health}"
HEALTH_CHECK_TIMEOUT=60
HEALTH_CHECK_INTERVAL=2
# UID/GID пользователя внутри образа (docker/Dockerfile: app = 1000:1000).
# Файлы на bind-mount ./data должны принадлежать ему, иначе SQLite не откроет БД.
CONTAINER_UID="${CONTAINER_UID:-1000}"
CONTAINER_GID="${CONTAINER_GID:-1000}"

# Source common functions if available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "${SCRIPT_DIR}/lib/common.sh" ]]; then
    source "${SCRIPT_DIR}/lib/common.sh"
else
    # Define basic logging functions if common.sh not available
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    NC='\033[0m'
    log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
    log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
    log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
    log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
fi

# Timestamp for backups
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
UPGRADE_BACKUP_DIR="${BACKUP_DIR}/upgrade_${TIMESTAMP}"
CURRENT_VERSION=""
# git-ref (тег, ветка или коммит), из которого пересобирается образ.
TARGET_VERSION="main"
ROLLBACK_ON_FAILURE=true
# Caddy перечитывает конфиг только по команде: bind-mount меняется молча.
CADDYFILE_CHANGED=false
# Коммит, из которого собран работающий образ (заполняется build_new_version).
PREVIOUS_HEAD=""

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --version)
                TARGET_VERSION="$2"
                shift 2
                ;;
            --no-rollback)
                ROLLBACK_ON_FAILURE=false
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            rollback)
                manual_rollback
                exit $?
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat <<EOF
Family Budget Service - Upgrade Script

Usage: sudo ./upgrade.sh [OPTIONS] [COMMAND]

OPTIONS:
    --version GIT_REF    Target git ref - tag, branch or commit (default: main).
                         The image is built from ${SRC_DIR}, there is no
                         published registry image to pull, so this is never an
                         image tag.
    --no-rollback        Disable automatic rollback on failure
    --help, -h           Show this help message

COMMANDS:
    rollback             Manually rollback to previous version

EXAMPLES:
    # Upgrade to the tip of the default branch
    sudo ./upgrade.sh

    # Upgrade to a specific git ref
    sudo ./upgrade.sh --version v1.2.3

    # Upgrade without auto-rollback
    sudo ./upgrade.sh --no-rollback

    # Manual rollback
    sudo ./upgrade.sh rollback

EOF
}

# ============================================
# Pre-upgrade Checks
# ============================================

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi
}

check_installation() {
    log_info "Checking installation..."

    if [[ ! -d "${INSTALL_DIR}" ]]; then
        log_error "Installation directory not found: ${INSTALL_DIR}"
        exit 1
    fi

    if [[ ! -f "${COMPOSE_FILE}" ]]; then
        log_error "Docker Compose file not found: ${COMPOSE_FILE}"
        log_info "Checked: ${COMPOSE_FILE}"
        exit 1
    fi

    if [[ ! -d "${SRC_DIR}/.git" ]]; then
        log_error "Source checkout not found: ${SRC_DIR}"
        log_error "The image is built from source; re-run install.sh to create it"
        exit 1
    fi

    log_success "Installation directory found"
}

check_disk_space() {
    log_info "Checking disk space..."

    local available
    available=$(df -BM "${INSTALL_DIR}" | tail -1 | awk '{print $4}' | sed 's/M//')
    local required=500  # 500MB minimum

    if [[ ${available} -lt ${required} ]]; then
        log_error "Insufficient disk space. Available: ${available}MB, Required: ${required}MB"
        exit 1
    fi

    log_success "Disk space OK (${available}MB available)"
}

get_current_version() {
    log_info "Getting current version..."

    # Версия — это коммит, из которого собран текущий образ.
    CURRENT_VERSION=$(git -C "${SRC_DIR}" rev-parse HEAD 2>/dev/null || echo "unknown")

    local human
    human=$(git -C "${SRC_DIR}" describe --tags --always 2>/dev/null || echo "${CURRENT_VERSION}")

    log_info "Current version: ${human} (${CURRENT_VERSION})"
}

check_database_integrity() {
    log_info "Checking database integrity..."

    local db_file="${DATA_DIR}/budget.db"

    if [[ ! -f "${db_file}" ]]; then
        log_warning "Database file not found, will be created on first run"
        return 0
    fi

    # Внутри рантайм-образа sqlite3 нет (docker/Dockerfile ставит только
    # ca-certificates/tzdata/wget), поэтому проверка делается на хосте.
    if ! command -v sqlite3 >/dev/null 2>&1; then
        log_warning "sqlite3 is not installed on the host, skipping integrity check"
        log_warning "Install it (apt-get install sqlite3 / dnf install sqlite) for safer upgrades"
        return 0
    fi

    local integrity
    integrity=$(sqlite3 "${db_file}" 'PRAGMA integrity_check;' 2>/dev/null || echo "cannot_check")

    if [[ "${integrity}" == "ok" ]]; then
        log_success "Database integrity: OK"
        return 0
    fi

    if [[ "${integrity}" == "cannot_check" ]]; then
        log_warning "Could not check database integrity"
        return 0
    fi

    log_error "Database integrity check failed: ${integrity}"
    exit 1
}

# ============================================
# Backup Functions
# ============================================

# Take a database copy with the application's own `backup` subcommand
# (VACUUM INTO, so a single self-contained file - no -wal/-shm to carry along).
#
# `docker compose run`, not `exec`: after a failed upgrade the app container may be
# missing entirely, while `run` works whether the service is up or down. --no-deps
# keeps compose from starting Caddy for a one-shot command.
create_database_backup() {
    local dest=$1

    cd "${INSTALL_DIR}" || return 1

    if ! docker compose -f "${COMPOSE_FILE}" run --rm --no-deps app backup; then
        log_error "The backup subcommand failed"
        return 1
    fi

    # The subcommand writes into BACKUP_DIR under its own timestamped name.
    local latest
    # shellcheck disable=SC2012 # имена файлов задаёт сам сервис: backup_<дата>_<время>.db
    latest=$(ls -t "${BACKUP_DIR}"/backup_*.db 2>/dev/null | head -1)
    if [[ -z "${latest}" ]]; then
        log_error "No backup file appeared in ${BACKUP_DIR}"
        return 1
    fi

    # Copied into the upgrade directory: retention (BACKUP_KEEP) prunes ${BACKUP_DIR},
    # and rollback must still find the file it was promised.
    cp "${latest}" "${dest}" || return 1
    log_info "Backup taken from ${latest}"
}

# Снять предобновленческий бэкап. Каждая операция проверяется явно и возвращает
# 1: `set -e` внутри функции, вызванной в условии (`if ! create_upgrade_backup`),
# не работает — без явных проверок
# частично снятый бэкап считался бы удачным, и откат восстанавливал бы обрывок.
create_upgrade_backup() {
    log_info "Creating pre-upgrade backup..."

    mkdir -p "${UPGRADE_BACKUP_DIR}" || return 1

    # Backup database if exists
    # Маркер пишется всегда: по одному лишь отсутствию снимка откат не отличит
    # «базы ещё не было» от «бэкап потерян», а это противоположные действия.
    local db_file="${DATA_DIR}/budget.db"
    if [[ -f "${db_file}" ]]; then
        log_info "Backing up database..."
        create_database_backup "${UPGRADE_BACKUP_DIR}/budget.db" || return 1
        echo "yes" > "${UPGRADE_BACKUP_DIR}/db_present.txt" || return 1
        log_success "Database backed up"
    else
        echo "no" > "${UPGRADE_BACKUP_DIR}/db_present.txt" || return 1
        log_info "No database at ${db_file}; nothing to back up"
    fi

    # Backup environment file
    if [[ -f "${INSTALL_DIR}/.env" ]]; then
        log_info "Backing up environment file..."
        cp "${INSTALL_DIR}/.env" "${UPGRADE_BACKUP_DIR}/.env" || return 1
    fi

    # Compose и Caddyfile обновляются вместе с образом (sync_deploy_files),
    # значит откат обязан вернуть и их.
    # Маркер caddy_present.txt — по той же причине, что и db_present.txt: установка
    # до перехода на Caddy идёт без Caddyfile, и откат не должен путать «его не было»
    # с «снимок потерян».
    cp "${COMPOSE_FILE}" "${UPGRADE_BACKUP_DIR}/docker-compose.yml" || return 1
    if [[ -f "${CADDYFILE}" ]]; then
        cp "${CADDYFILE}" "${UPGRADE_BACKUP_DIR}/Caddyfile" || return 1
        echo "yes" > "${UPGRADE_BACKUP_DIR}/caddy_present.txt" || return 1
    else
        echo "no" > "${UPGRADE_BACKUP_DIR}/caddy_present.txt" || return 1
    fi

    # Save current version info
    echo "${CURRENT_VERSION}" > "${UPGRADE_BACKUP_DIR}/version.txt" || return 1
    date > "${UPGRADE_BACKUP_DIR}/backup_time.txt" || return 1

    docker inspect family-budget-app > "${UPGRADE_BACKUP_DIR}/container_info.json" 2>/dev/null || true

    log_success "Pre-upgrade backup created: ${UPGRADE_BACKUP_DIR}"
}

# ============================================
# Upgrade Functions
# ============================================

# Вернуть рабочее дерево на коммит, из которого собран текущий (уже запущенный)
# образ. Без этого неудачная сборка оставляет ${SRC_DIR} на новом коммите, тогда
# как контейнер продолжает работать на старом образе: get_current_version на
# следующем запуске отдаёт версию, которая никогда не разворачивалась, проверка
# "CURRENT_VERSION == TARGET_VERSION" пропускает нужную пересборку, а version.txt
# в бэкапе фиксирует сломанную ревизию — то есть откат восстанавливает не то.
restore_src_checkout() {
    local commit=$1

    if [[ -z "${commit}" || "${commit}" == "unknown" ]]; then
        log_error "Previous commit is unknown, leaving ${SRC_DIR} as is"
        return 1
    fi

    if ! git -C "${SRC_DIR}" checkout --force --detach "${commit}" 2>/dev/null; then
        log_error "Failed to restore sources to ${commit}; ${SRC_DIR} is out of sync with the running image"
        return 1
    fi

    log_info "Restored sources to ${commit}"
}

# Топология (compose, Caddyfile) живёт в ${INSTALL_DIR}, а не в образе: без этой
# синхронизации новый релиз запускался бы на compose предыдущего — без новых
# переменных, монтирований и с прежним digest'ом Caddy.
sync_deploy_files() {
    local compose_src="${SRC_DIR}/deploy/docker-compose.yml"
    local caddy_src="${SRC_DIR}/deploy/caddy/Caddyfile"

    if [[ ! -f "${compose_src}" || ! -f "${caddy_src}" ]]; then
        log_error "Deployment files not found under ${SRC_DIR}/deploy"
        return 1
    fi

    cp "${compose_src}" "${COMPOSE_FILE}" || return 1
    mkdir -p "$(dirname "${CADDYFILE}")" || return 1
    if ! cmp -s "${caddy_src}" "${CADDYFILE}"; then
        CADDYFILE_CHANGED=true
    fi
    cp "${caddy_src}" "${CADDYFILE}" || return 1

    log_info "Deployment files synced from ${SRC_DIR}/deploy"
}

# Вернуть compose и Caddyfile из предобновленческого бэкапа.
# Пропавший снимок — отказ, а не «нечего восстанавливать»: compose снимается
# всегда, поэтому его отсутствие (старый каталог бэкапа под `rollback`, обрыв
# на полпути) означает, что топология осталась от целевой версии, и следующий
# запуск примет её за развёрнутую. Про Caddyfile решает маркер caddy_present.txt:
# если его не было, файл неудачного обновления удаляется — восстановленный
# compose о нём не знает.
restore_deploy_files() {
    if [[ ! -f "${UPGRADE_BACKUP_DIR}/docker-compose.yml" ]]; then
        log_error "No compose snapshot in ${UPGRADE_BACKUP_DIR}; the topology cannot be rolled back"
        return 1
    fi
    if ! cp "${UPGRADE_BACKUP_DIR}/docker-compose.yml" "${COMPOSE_FILE}"; then
        log_error "Failed to restore ${COMPOSE_FILE} from ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    local caddy_present
    caddy_present=$(cat "${UPGRADE_BACKUP_DIR}/caddy_present.txt" 2>/dev/null || echo "unknown")

    case "${caddy_present}" in
        yes)
            if [[ ! -f "${UPGRADE_BACKUP_DIR}/Caddyfile" ]]; then
                log_error "A Caddyfile existed before the upgrade, but its snapshot is missing in ${UPGRADE_BACKUP_DIR}"
                return 1
            fi
            if ! cmp -s "${UPGRADE_BACKUP_DIR}/Caddyfile" "${CADDYFILE}"; then
                CADDYFILE_CHANGED=true
            fi
            if ! cp "${UPGRADE_BACKUP_DIR}/Caddyfile" "${CADDYFILE}"; then
                log_error "Failed to restore ${CADDYFILE} from ${UPGRADE_BACKUP_DIR}"
                return 1
            fi
            ;;
        no)
            if [[ -e "${CADDYFILE}" ]] && ! rm -f "${CADDYFILE}"; then
                log_error "Could not remove the Caddyfile created by the failed upgrade: ${CADDYFILE}"
                return 1
            fi
            # sync_deploy_files взвёл флаг, когда клал Caddyfile в установку без него.
            # Восстановленный compose сервиса caddy не знает, и reload_caddy утонул бы
            # на `compose run caddy`, объявив удавшийся откат неудачным.
            CADDYFILE_CHANGED=false
            ;;
        *)
            log_error "Backup ${UPGRADE_BACKUP_DIR} does not record whether a Caddyfile existed before the upgrade"
            log_error "Refusing to guess: check ${CADDYFILE} manually"
            return 1
            ;;
    esac
}

# Полный откат состояния на диске к работающей версии. Возвращает 1, если хоть
# одна половина не восстановлена: вызывающий не вправе сообщать об удачном откате.
restore_running_state() {
    local rc=0

    restore_deploy_files || rc=1
    restore_src_checkout "${PREVIOUS_HEAD}" || rc=1

    return ${rc}
}

# Сообщить оператору, что откат к работающей версии не удался.
log_restore_failure() {
    log_error "FAILED to restore the running version on disk"
    log_error "${SRC_DIR} and/or ${INSTALL_DIR} may still hold ${TARGET_VERSION}"
    log_error "Restore them from ${UPGRADE_BACKUP_DIR} before running the upgrade again"
}

# Перечитать конфиг Caddy, если он изменился: `up -d` пересоздаёт контейнер по
# изменению сервиса в compose, а не содержимого примонтированного файла.
reload_caddy() {
    [[ "${CADDYFILE_CHANGED}" == "true" ]] || return 0

    cd "${INSTALL_DIR}" || return 1

    # Валидация до применения: битый конфиг иначе доходит до restart, а тот
    # отдаёт успех и на контейнере, который вышел сразу после старта.
    # Одноразовый контейнер, а не `exec`: Caddy может и не работать.
    if ! docker compose -f "${COMPOSE_FILE}" run --rm --no-deps --entrypoint caddy \
        caddy validate --config /etc/caddy/Caddyfile >/dev/null 2>&1; then
        log_error "New Caddyfile is invalid, not applying it: ${CADDYFILE}"
        return 1
    fi

    if docker compose -f "${COMPOSE_FILE}" exec -T caddy \
        caddy reload --config /etc/caddy/Caddyfile >/dev/null 2>&1; then
        log_success "Caddy config reloaded"
        return 0
    fi

    log_warning "Caddy config reload failed, restarting Caddy..."
    if ! docker compose -f "${COMPOSE_FILE}" restart caddy; then
        log_error "Caddy restart failed; check ${CADDYFILE}"
        return 1
    fi

    sleep 5
    if [[ "$(container_state caddy)" != "running" ]]; then
        log_error "Caddy is not running after the restart; check ${CADDYFILE}"
        return 1
    fi

    log_success "Caddy restarted"
    return 0
}

# Обновить исходники и пересобрать образ.
# `docker compose pull` тут неприменим: сервис build-only (D-02).
build_new_version() {
    log_info "Fetching sources for: ${TARGET_VERSION}..."

    # Коммит, из которого собран работающий образ: на него откатываем дерево,
    # если что-то ниже (вплоть до stop_service) сломается.
    PREVIOUS_HEAD=$(git -C "${SRC_DIR}" rev-parse HEAD 2>/dev/null || echo "unknown")

    if ! git -C "${SRC_DIR}" fetch --tags --force origin "${TARGET_VERSION}"; then
        log_error "Failed to fetch ref: ${TARGET_VERSION}"
        return 1
    fi

    if ! git -C "${SRC_DIR}" checkout --force --detach FETCH_HEAD; then
        log_error "Failed to check out ref: ${TARGET_VERSION}"
        restore_src_checkout "${PREVIOUS_HEAD}"
        return 1
    fi

    if ! sync_deploy_files; then
        restore_running_state || log_restore_failure
        return 1
    fi

    log_info "Building image from $(git -C "${SRC_DIR}" rev-parse --short HEAD)..."

    cd "${INSTALL_DIR}"

    if VERSION="$(git -C "${SRC_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)" \
        docker compose -f "${COMPOSE_FILE}" build app; then
        log_success "Successfully built version: ${TARGET_VERSION}"
    else
        log_error "Failed to build new version"
        # Дерево возвращаем к работающему образу: иначе следующий запуск сочтёт
        # несобранный коммит текущей версией.
        restore_running_state || log_restore_failure
        return 1
    fi
}

stop_service() {
    log_info "Stopping service gracefully..."

    cd "${INSTALL_DIR}"

    if docker compose -f "${COMPOSE_FILE}" stop app 2>/dev/null; then
        log_success "Service stopped"
    else
        log_warning "Failed to stop service gracefully, forcing stop..."
        docker compose -f "${COMPOSE_FILE}" down 2>/dev/null || true
    fi

    # Wait for graceful shutdown
    sleep 5

    if ! app_container_stopped; then
        log_error "app container is still running (or its state is unknown) after stop; aborting the upgrade"
        log_error "Check: docker compose -f ${COMPOSE_FILE} ps"
        # Дерево и deploy-файлы уже переключены на целевую версию, а работает
        # прежний образ: без отката следующий запуск сочтёт цель развёрнутой.
        if restore_running_state; then
            log_error "Sources and deploy files restored to the running version; the database was not touched"
        else
            log_restore_failure
            log_error "The database was not touched"
        fi
        log_warning "The image built for ${TARGET_VERSION} stays in the local docker cache"
        exit 1
    fi
}

# Состояние контейнера сервиса: running | stopped | unknown.
# unknown — docker не ответил; отличать его обязательно: ошибка API, принятая
# за "остановлен", открывает дорогу перезаписи живой базы.
container_state() {
    local service=$1 cid state rc

    if ! cid="$(docker compose -f "${COMPOSE_FILE}" ps -aq "${service}" 2>/dev/null)"; then
        echo "unknown"
        return 0
    fi
    cid="${cid%%$'\n'*}"
    if [[ -z "${cid}" ]]; then
        echo "stopped"
        return 0
    fi

    state="$(docker inspect -f '{{.State.Running}}' "${cid}" 2>&1)"
    rc=$?
    if (( rc != 0 )); then
        # Контейнер удалён между ps и inspect — это "остановлен"; недоступный
        # демон или таймаут API состоянием считать нельзя.
        if [[ "${state}" == *"No such object"* ]]; then
            echo "stopped"
        else
            echo "unknown"
        fi
        return 0
    fi

    if [[ "${state}" == "true" ]]; then
        echo "running"
    else
        echo "stopped"
    fi
}

# Перед перезаписью budget.db проверка обязательна: копия поверх открытой
# SQLite-базы (и удалённый под ней -wal) её разрушает, поэтому неизвестное
# состояние приравнивается к работающему контейнеру.
app_container_stopped() {
    [[ "$(container_state app)" == "stopped" ]]
}

start_service() {
    log_info "Starting service with new version..."

    cd "${INSTALL_DIR}" || return 1

    # Весь проект, а не `up -d app`: аварийная ветка stop_service гасит стек
    # целиком (`down`), а Caddy зависит от app, не наоборот — с `up -d app` он
    # так и остался бы лежать, и наружу никто бы не отвечал.
    #
    # Явная проверка обязательна: `up -d` падает на занятом порте, на нехватке
    # места и на неверном .env, а вызывающая сторона обрабатывает только
    # ненулевой код возврата (внутри `if !` механизм `set -e` отключён).
    docker compose -f "${COMPOSE_FILE}" up -d || return 1

    # Wait for startup
    log_info "Waiting for service to start..."
    sleep 10
}

# Порт 8080 наружу не публикуется (наружу смотрит только Caddy), поэтому
# /health опрашивается изнутри контейнера — wget есть в рантайм-образе.
verify_health() {
    log_info "Verifying service health..."

    local elapsed=0
    local max_wait=${HEALTH_CHECK_TIMEOUT}

    cd "${INSTALL_DIR}" || return 1

    while [[ ${elapsed} -lt ${max_wait} ]]; do
        if docker compose -f "${COMPOSE_FILE}" exec -T app \
            wget -q -O /dev/null "${HEALTH_URL}" 2>/dev/null; then
            log_success "Health check passed!"
            return 0
        fi

        log_info "Health check attempt (${elapsed}s/${max_wait}s)"
        sleep ${HEALTH_CHECK_INTERVAL}
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
    done

    log_error "Health check failed after ${max_wait} seconds"
    return 1
}

# ============================================
# Rollback Functions
# ============================================

# Откатить исходники на сохранённый коммит.
# install.sh клонирует с `--depth 1`, поэтому предыдущего коммита в истории
# может не быть вовсе — сначала углубляем историю, и только потом checkout.
restore_source_checkout() {
    local commit=$1

    if ! git -C "${SRC_DIR}" cat-file -e "${commit}^{commit}" 2>/dev/null; then
        log_info "Commit ${commit} is not in the local history, deepening the clone..."
        if [[ "$(git -C "${SRC_DIR}" rev-parse --is-shallow-repository 2>/dev/null)" == "true" ]]; then
            git -C "${SRC_DIR}" fetch --unshallow origin 2>/dev/null \
                || git -C "${SRC_DIR}" fetch --deepen=50 origin 2>/dev/null \
                || true
        fi
        # прицельная попытка: часть серверов отдаёт коммит по SHA
        git -C "${SRC_DIR}" fetch origin "${commit}" 2>/dev/null || true
    fi

    if ! git -C "${SRC_DIR}" cat-file -e "${commit}^{commit}" 2>/dev/null; then
        log_error "Commit ${commit} is still unreachable in ${SRC_DIR}"
        return 1
    fi

    git -C "${SRC_DIR}" checkout --force --detach "${commit}"
}

# Вернуть базу в предобновленческое состояние. Решение принимается по маркеру
# db_present.txt, а не по наличию снимка: если базы не было, её мог создать и
# смигрировать неудачный запуск, и старая версия поднялась бы на чужой схеме —
# такую базу надо удалить, а не оставить. Пропавший снимок при db_present=yes
# останавливает откат: восстанавливать нечем.
# Осиротевшие -wal/-shm удаляются перед подменой файла — SQLite накатил бы их
# поверх восстановленной базы и вернул данные неудачного обновления обратно.
restore_database() {
    local db_present
    db_present=$(cat "${UPGRADE_BACKUP_DIR}/db_present.txt" 2>/dev/null || echo "unknown")

    case "${db_present}" in
        yes)
            if [[ ! -f "${UPGRADE_BACKUP_DIR}/budget.db" ]]; then
                log_error "The database existed before the upgrade, but its snapshot is missing in ${UPGRADE_BACKUP_DIR}"
                log_error "Refusing to start the previous version over the failed upgrade's database"
                return 1
            fi
            log_info "Restoring database from backup..."
            if ! rm -f "${DATA_DIR}/budget.db-wal" "${DATA_DIR}/budget.db-shm"; then
                log_error "Could not remove stale WAL/SHM files in ${DATA_DIR}"
                return 1
            fi
            if ! cp "${UPGRADE_BACKUP_DIR}/budget.db" "${DATA_DIR}/budget.db"; then
                log_error "Restoring the database failed; ${DATA_DIR}/budget.db may be incomplete"
                return 1
            fi

            # Скрипт работает из-под root, а контейнер запущен как uid 1000: без chown
            # откатанный сервис не откроет восстановленную базу на запись.
            chown "${CONTAINER_UID}:${CONTAINER_GID}" "${DATA_DIR}/budget.db" 2>/dev/null || true

            log_success "Database restored"
            ;;
        no)
            if [[ -e "${DATA_DIR}/budget.db" || -e "${DATA_DIR}/budget.db-wal" ]]; then
                log_info "No database existed before the upgrade; removing the one it created..."
                if ! rm -f "${DATA_DIR}/budget.db" "${DATA_DIR}/budget.db-wal" "${DATA_DIR}/budget.db-shm"; then
                    log_error "Could not remove the database created by the failed upgrade in ${DATA_DIR}"
                    return 1
                fi
            fi
            log_success "Database returned to its pre-upgrade state (absent)"
            ;;
        *)
            log_error "Backup ${UPGRADE_BACKUP_DIR} does not record whether the database existed before the upgrade"
            log_error "Refusing to guess: check ${DATA_DIR}/budget.db manually"
            return 1
            ;;
    esac
}

rollback() {
    log_error "Upgrade failed, initiating automatic rollback..."

    if [[ ! -d "${UPGRADE_BACKUP_DIR}" ]]; then
        log_error "Backup directory not found, cannot rollback: ${UPGRADE_BACKUP_DIR}"
        log_error "Manual intervention required"
        return 1
    fi

    # Stop current service
    log_info "Stopping failed upgrade..."
    cd "${INSTALL_DIR}"
    if ! docker compose -f "${COMPOSE_FILE}" stop app 2>/dev/null; then
        log_warning "Graceful stop failed, killing the container..."
        docker compose -f "${COMPOSE_FILE}" kill app 2>/dev/null || true
    fi
    sleep 3

    if ! app_container_stopped; then
        log_error "app container is still running (or its state is unknown); refusing to overwrite the live database"
        log_error "Stop it manually and restore from ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    if ! restore_database; then
        log_error "The service is left stopped. Restore manually from ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    # Restore environment files
    if [[ -f "${UPGRADE_BACKUP_DIR}/.env" ]]; then
        if ! cp "${UPGRADE_BACKUP_DIR}/.env" "${INSTALL_DIR}/.env"; then
            log_error "Restoring .env failed; the service would start on the failed upgrade's config"
            log_error "The service is left stopped. Restore manually from ${UPGRADE_BACKUP_DIR}"
            return 1
        fi
    fi

    local deploy_restored=true
    restore_deploy_files || deploy_restored=false

    # Restore previous version: откатываем исходники на сохранённый коммит и
    # пересобираем образ — готового образа для отката в реестре нет (D-02).
    # Если бэкап не успел записать version.txt (падение до этого шага), берём
    # версию, определённую в начале прогона: она и есть уже развёрнутая.
    local prev_version
    prev_version=$(cat "${UPGRADE_BACKUP_DIR}/version.txt" 2>/dev/null || echo "")
    if [[ -z "${prev_version}" ]]; then
        prev_version="${CURRENT_VERSION:-unknown}"
    fi
    log_info "Restoring previous version: ${prev_version}"

    # Отдельный флаг: /health отвечает и у неудачного обновления, поэтому «сервис
    # поднялся» не означает «откатились». Ниже он решает, запускать ли сервис
    # вообще.
    local image_restored=true

    if [[ "${prev_version}" == "unknown" ]]; then
        image_restored=false
        log_error "Previous commit is unknown; restarting with the image that is already built"
        log_error "(that image is the FAILED upgrade — verify the service manually)"
    elif restore_source_checkout "${prev_version}"; then
        if ! VERSION="$(git -C "${SRC_DIR}" describe --tags --always --dirty 2>/dev/null || echo dev)" \
            docker compose -f "${COMPOSE_FILE}" build app; then
            image_restored=false
            log_error "Rebuild of previous version failed; the image still contains the failed upgrade"
        fi
    else
        image_restored=false
        log_error "Could not restore sources at ${prev_version}; the image is NOT rebuilt"
        log_error "and still contains the failed upgrade. Restore manually from ${UPGRADE_BACKUP_DIR}"
    fi

    # Проверки до `up -d`, а не после: образ неудачного обновления, поднятый над
    # уже восстановленной базой, накатит на неё свои миграции — это уничтожает
    # результат отката. Сервис остаётся погашенным, пока оператор не вмешается.
    if [[ "${deploy_restored}" != "true" ]]; then
        log_error "Compose/Caddyfile were not fully restored from ${UPGRADE_BACKUP_DIR}"
        log_error "The service is left STOPPED: starting it would run the failed upgrade's topology"
        log_error "against the restored database. Fix it manually."
        return 1
    fi

    if [[ "${image_restored}" != "true" ]]; then
        log_error "Only the database and the config were rolled back: the built image is"
        log_error "STILL the failed upgrade."
        log_error "The service is left STOPPED: starting that image would write (and migrate) the"
        log_error "restored database. Fix it manually."
        log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    # --remove-orphans: откат на топологию без Caddy (caddy_present=no) оставляет
    # контейнер Caddy от неудачного обновления жить на 80/443 — с уже удалённым
    # конфигом и без апстрима, то есть публичный вход отдаёт 502 при «успешном» откате.
    if ! docker compose -f "${COMPOSE_FILE}" up -d --remove-orphans; then
        log_error "Rollback failed: the service did not start. Manual intervention required!"
        log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    sleep 10

    if ! verify_health; then
        log_error "Rollback failed! Manual intervention required!"
        log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    if ! reload_caddy; then
        log_error "Rollback started the service, but Caddy did not pick up the restored config"
        log_error "Public ingress may be down. Backup location: ${UPGRADE_BACKUP_DIR}"
        return 1
    fi

    log_success "Rollback successful! Service is running on previous version: ${prev_version}"
    return 0
}

# Единая точка обработки провала обновления после остановки сервиса.
#
# Раньше её не было: в rollback уходил только провал verify_health, а падение
# `docker compose up -d` (диск, порт, битый .env) просто завершало скрипт по
# `set -e` — сервис оставался погашенным, откат не выполнялся, и оператор не
# получал даже инструкций по восстановлению.
handle_upgrade_failure() {
    local reason=$1

    # Снимаем ловушку сразу: внутри rollback команды падают штатно
    # (`|| true`, отсутствующие файлы), рекурсивный вход всё бы испортил.
    trap - ERR

    log_error "Upgrade failed: ${reason}"

    if [[ "${ROLLBACK_ON_FAILURE}" == "true" ]]; then
        if rollback; then
            log_warning "Upgrade was rolled back successfully"
            exit 1
        fi

        log_error "Rollback failed!"
        log_error "Manual recovery required"
        log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
        exit 2
    fi

    log_error "Auto-rollback disabled, manual intervention required"
    log_error "The service is currently STOPPED"
    log_error "To rollback manually, run: $0 rollback"
    log_error "Backup location: ${UPGRADE_BACKUP_DIR}"
    exit 1
}

manual_rollback() {
    log_info "=== Manual Rollback ==="

    # Find most recent upgrade backup
    local latest_backup
    # shellcheck disable=SC2012 # каталоги создаёт этот же скрипт: upgrade_<дата>_<время>
    latest_backup=$(ls -td "${BACKUP_DIR}"/upgrade_* 2>/dev/null | head -1)
    
    if [[ -z "${latest_backup}" ]]; then
        log_error "No upgrade backups found in ${BACKUP_DIR}"
        exit 1
    fi
    
    log_info "Found backup: ${latest_backup}"
    UPGRADE_BACKUP_DIR="${latest_backup}"
    
    rollback
}

# ============================================
# Main Upgrade Function
# ============================================

upgrade() {
    echo ""
    log_info "╔════════════════════════════════════════════════════════════════╗"
    log_info "║       Family Budget Service - Upgrade Process                 ║"
    log_info "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Target version: ${TARGET_VERSION}"
    log_info "Backup directory: ${UPGRADE_BACKUP_DIR}"
    log_info "Auto-rollback: ${ROLLBACK_ON_FAILURE}"
    echo ""

    # Pre-flight checks
    check_root
    check_installation
    check_disk_space
    get_current_version

    # Skip if the requested ref is already checked out (полный SHA)
    if [[ "${CURRENT_VERSION}" == "${TARGET_VERSION}" ]]; then
        log_info "Already running ${TARGET_VERSION}"
        log_info "No upgrade needed"
        exit 0
    fi

    check_database_integrity

    # Бэкап снимается ДО сборки: подкоманда `backup` открывает БД через
    # OpenDatabase, а тот накатывает миграции. Снятый новым образом
    # «предобновленческий» бэкап уже содержал бы новую схему, и откатывать было
    # бы не на что. VACUUM INTO даёт консистентный снимок и на живом сервисе.
    if ! create_upgrade_backup; then
        log_error "Failed to create the pre-upgrade backup"
        log_error "Nothing has been changed; the service keeps running the current version"
        exit 1
    fi

    # Fetch sources and rebuild the image: неудачная сборка ничего не меняет на диске.
    if ! build_new_version; then
        log_error "Failed to build new version"
        exit 1
    fi

    # Stop service
    stop_service

    # Начиная отсюда сервис остановлен: любая ошибка обязана уходить в
    # handle_upgrade_failure, иначе `set -e` завершит скрипт с погашенным
    # сервисом и без единой попытки отката (ROLLBACK_ON_FAILURE игнорируется).
    trap 'handle_upgrade_failure "unexpected error at line ${LINENO}"' ERR

    # Start with new version
    if ! start_service; then
        handle_upgrade_failure "failed to start the service with the new version"
    fi

    # Verify health
    if verify_health; then
        trap - ERR
        # Откат сюда не годится: приложение уже работает на новой версии, а лежит
        # только ingress — чинится он перезапуском Caddy, а не возвратом схемы.
        if ! reload_caddy; then
            log_error "App upgraded to ${TARGET_VERSION}, but Caddy did not pick up the new config"
            log_error "Public ingress may be down; check: docker compose -f ${COMPOSE_FILE} ps caddy"
            exit 1
        fi
        log_success "╔════════════════════════════════════════════════════════════════╗"
        log_success "║       Upgrade Successful!                                      ║"
        log_success "╚════════════════════════════════════════════════════════════════╝"
        echo ""
        log_info "Upgraded from: ${CURRENT_VERSION}"
        log_info "Upgraded to: ${TARGET_VERSION}"
        log_info "Backup location: ${UPGRADE_BACKUP_DIR}"
        log_info "Health check: PASSED"
        echo ""
        log_info "Service is running behind Caddy"
        echo ""
        exit 0
    else
        handle_upgrade_failure "health check failed after upgrade"
    fi
}

# ============================================
# Entry Point
# ============================================

main() {
    parse_args "$@"
    upgrade
}

main "$@"
