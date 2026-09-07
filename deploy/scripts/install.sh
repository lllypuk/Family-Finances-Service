#!/bin/bash
# Family Budget Service - installer for the single-host deployment (app + Caddy in one compose).

set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/family-budget}"
# Образ приезжает из реестра GitLab: на сервере ничего не собирается.
# Реестр закрытый, поэтому до первого запуска нужен `docker login`.
DEFAULT_IMAGE="registry.gitlab.shatrov.tech/shatrov.tech/family-finances-service:main"
LOG_FILE="${LOG_FILE:-/var/log/family-budget-install.log}"
DEFAULT_DOMAIN="ffs.shatrov.tech"

# UID/GID of the user inside the image (docker/Dockerfile: USER 1000:1000).
# The bind-mounted data/ and backups/ must belong to it or SQLite cannot open the database (D-03).
CONTAINER_UID=1000
CONTAINER_GID=1000

# Network left by an installation older than this script: it sits on the same
# subnet as the new one, so `up` fails with "Pool overlaps" until it is gone.
LEGACY_NETWORK="family-budget_family-budget-net"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=./lib/docker.sh
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck source=./lib/firewall.sh
source "${SCRIPT_DIR}/lib/firewall.sh"

DOMAIN=""
ACME_EMAIL=""
IMAGE=""
NON_INTERACTIVE=false
DRY_RUN=false
# --reinstall: explicit consent to wipe an existing installation (the whole tree,
# database included, is moved to ${INSTALL_DIR}.backup.<ts>). Without it a repeated
# run keeps data/, backups/ and .env.
REINSTALL=false
# Каталог с файлами развёртывания: этот скрипт лежит в deploy/scripts/ того же
# клона, из которого его и запускают.
DEPLOY_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
# Повторный запуск переписывает Caddyfile у уже работающей установки; set by copy_deploy_files.
CADDYFILE_CHANGED=false
# Копия работающего Caddyfile на время замены (см. copy_deploy_files/reload_caddy).
CADDYFILE_BACKUP="${INSTALL_DIR}/caddy/Caddyfile.prev"

# In --dry-run every mutating command is printed instead of executed.
run() {
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] $*"
        return 0
    fi
    "$@"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --image)
                IMAGE="$2"
                shift 2
                ;;
            --non-interactive)
                NON_INTERACTIVE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                NON_INTERACTIVE=true
                shift
                ;;
            --reinstall)
                REINSTALL=true
                shift
                ;;
            --domain)
                DOMAIN="$2"
                shift 2
                ;;
            --email)
                ACME_EMAIL="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                exit 0
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
Family Budget Service - Installation Script

Usage: sudo ./install.sh [OPTIONS]

OPTIONS:
    --domain DOMAIN     Domain served by Caddy (default: ${DEFAULT_DOMAIN})
    --email EMAIL       Contact e-mail for Let's Encrypt (default: admin@DOMAIN)
    --image REF         Application image to pull (default: ${DEFAULT_IMAGE})
    --non-interactive   Run without prompts
    --dry-run           Print every mutating command instead of running it
    --reinstall         DESTRUCTIVE. Move the existing ${INSTALL_DIR} aside
                        (database and backups included) and install from scratch
    --help, -h          Show this help message

EXAMPLES:
    sudo ./install.sh --domain ${DEFAULT_DOMAIN} --email admin@example.com
    sudo ./install.sh --non-interactive

ENVIRONMENT:
    INSTALL_DIR         Installation directory (default: /opt/family-budget)

REQUIREMENTS:
    - Ubuntu 22.04/24.04, Debian 11/12, or Rocky Linux 9
    - 512MB RAM, 5GB disk, root privileges, ports 80/443 free
    - \`docker login\` for the image registry: it is private, and the pull below
      fails without it
    - DNS A record pointing at this host: Caddy issues the certificate on first start
    - Run this script from a checkout: the compose files and the Caddyfile are
      taken from ../ relative to it, nothing is cloned
EOF
}

prompt_configuration() {
    # Дефолты берутся из уже установленного .env: запуск без --domain не должен
    # молча вернуть домен к DEFAULT_DOMAIN, а с --domain — обязан его сменить.
    local env_file="${INSTALL_DIR}/.env"
    if [[ -f "${env_file}" && "${REINSTALL}" != "true" ]]; then
        DOMAIN="${DOMAIN:-$(env_value "${env_file}" DOMAIN)}"
        ACME_EMAIL="${ACME_EMAIL:-$(env_value "${env_file}" ACME_EMAIL)}"
        IMAGE="${IMAGE:-$(env_value "${env_file}" FFS_IMAGE)}"
    fi

    if [[ "${NON_INTERACTIVE}" != "true" ]]; then
        echo ""
        log_info "=== Configuration ==="
        prompt_input "Domain served by Caddy" "${DOMAIN:-${DEFAULT_DOMAIN}}" DOMAIN
        prompt_input "Contact e-mail for Let's Encrypt" "${ACME_EMAIL:-admin@${DOMAIN}}" ACME_EMAIL
    fi

    DOMAIN="${DOMAIN:-${DEFAULT_DOMAIN}}"
    ACME_EMAIL="${ACME_EMAIL:-admin@${DOMAIN}}"
    IMAGE="${IMAGE:-${DEFAULT_IMAGE}}"

    log_info "Domain: ${DOMAIN}"
    log_info "ACME e-mail: ${ACME_EMAIL}"
    log_info "Image: ${IMAGE}"
    log_info "Install directory: ${INSTALL_DIR}"

    if ! confirm_action "Proceed with installation?"; then
        log_info "Installation cancelled by user"
        exit 0
    fi
}

create_directories() {
    log_info "Creating directory structure..."

    if [[ -d "${INSTALL_DIR}" && "${REINSTALL}" == "true" ]]; then
        log_warning "--reinstall: moving the existing installation aside"
        # The container holds the database file open on the bind mount: stop it first.
        if [[ -f "${INSTALL_DIR}/docker-compose.yml" ]]; then
            run bash -c "cd '${INSTALL_DIR}' && docker compose down 2>/dev/null" || true
        fi
        run backup_directory "${INSTALL_DIR}"
    elif [[ -d "${INSTALL_DIR}" ]]; then
        log_info "Existing installation found - keeping data/, backups/ and .env"
        log_info "Pass --reinstall to wipe it (the old tree moves to ${INSTALL_DIR}.backup.<timestamp>)"
    fi

    run mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/backups" "${INSTALL_DIR}/caddy" "${INSTALL_DIR}/logs/caddy"
    log_success "Directory structure ready at ${INSTALL_DIR}"
}

# There is no published image: the image is built on this host, so the sources
# have to sit next to the compose file (compose sets BUILD_CONTEXT=./src).
check_deploy_dir() {
    local missing=0
    local f
    for f in docker-compose.yml docker-compose.proxied.yml caddy/Caddyfile .env.example release.sh; do
        [[ -f "${DEPLOY_DIR}/${f}" ]] || { log_error "Missing ${DEPLOY_DIR}/${f}"; missing=1; }
    done
    if [[ ${missing} -ne 0 ]]; then
        log_error "Run this script from a complete checkout: deploy/scripts/install.sh"
        exit 1
    fi
    log_info "Deployment files: ${DEPLOY_DIR}"
}

copy_deploy_files() {
    log_info "Copying deployment files..."

    local compose_src="${DEPLOY_DIR}/docker-compose.yml"
    local caddyfile_src="${DEPLOY_DIR}/caddy/Caddyfile"

    if [[ ! -f "${compose_src}" || ! -f "${caddyfile_src}" ]]; then
        log_error "Deployment files not found under ${DEPLOY_DIR}"
        log_error "The checkout is incomplete (wrong REPO_GIT_URL/REPO_REF?)"
        exit 1
    fi

    if [[ -f "${INSTALL_DIR}/caddy/Caddyfile" ]] && ! cmp -s "${caddyfile_src}" "${INSTALL_DIR}/caddy/Caddyfile"; then
        CADDYFILE_CHANGED=true
        # Новый конфиг проверяется только после того, как ляжет на место; без
        # этой копии невалидный файл остался бы примонтированным и убил бы Caddy
        # на первом же пересоздании контейнера.
        run cp "${INSTALL_DIR}/caddy/Caddyfile" "${CADDYFILE_BACKUP}"
    fi

    run cp "${compose_src}" "${INSTALL_DIR}/docker-compose.yml"
    # Оверлей и release.sh кладутся всегда: ими пользуется выкат из CI, а
    # раскладку машины выбирает COMPOSE_FILE в .env, а не набор файлов.
    run cp "${DEPLOY_DIR}/docker-compose.proxied.yml" "${INSTALL_DIR}/docker-compose.proxied.yml"
    run cp "${DEPLOY_DIR}/release.sh" "${INSTALL_DIR}/release.sh"
    run cp "${caddyfile_src}" "${INSTALL_DIR}/caddy/Caddyfile"
    log_success "Deployment files copied"
}

# Перечитать конфиг Caddy, если файл заменён на повторном запуске: `up -d`
# пересоздаёт контейнер по изменению сервиса в compose, а не содержимого
# примонтированного файла.
reload_caddy() {
    [[ "${CADDYFILE_CHANGED}" == "true" ]] || return 0
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] would reload the Caddy config"
        return 0
    fi

    log_info "Caddyfile changed, reloading Caddy..."

    # Валидация до применения: битый конфиг иначе доходит до restart, а тот
    # отдаёт успех и на контейнере, который вышел сразу после старта.
    if ! docker compose run --rm --no-deps --entrypoint caddy \
        caddy validate --config /etc/caddy/Caddyfile >/dev/null 2>&1; then
        log_error "New Caddyfile is invalid: ${INSTALL_DIR}/caddy/Caddyfile"
        if [[ -f "${CADDYFILE_BACKUP}" ]]; then
            cp "${CADDYFILE_BACKUP}" "${INSTALL_DIR}/caddy/Caddyfile"
            log_error "Restored the previous config; Caddy keeps serving it"
        fi
        exit 1
    fi

    rm -f "${CADDYFILE_BACKUP}"

    if docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile >/dev/null 2>&1; then
        log_success "Caddy config reloaded"
        return 0
    fi

    log_warning "Caddy config reload failed, restarting Caddy..."
    if ! docker compose restart caddy; then
        log_error "Caddy restart failed; check ${INSTALL_DIR}/caddy/Caddyfile"
        exit 1
    fi

    log_success "Caddy restarted"
}

# Значение ключа в env-файле; пусто, если ключа (или файла) нет.
env_value() {
    local file=$1 key=$2

    [[ -f "${file}" ]] || return 0
    sed -n "s|^${key}=||p" "${file}" | tail -1
}

# Дописать ключ, если его в файле нет; существующее значение не трогаем.
ensure_env_key() {
    local file=$1 key=$2 value=$3

    if grep -q "^${key}=" "${file}"; then
        return 0
    fi
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] would append ${key}=${value} to ${file}"
        return 0
    fi

    log_info "Adding the missing ${key} to ${file}"
    # Файл без завершающего перевода строки склеил бы новый ключ с последним.
    if [[ -s "${file}" && -n "$(tail -c1 "${file}")" ]]; then
        echo >> "${file}"
    fi
    echo "${key}=${value}" >> "${file}"
}

# Записать ключ, переписав отличающееся значение: домен и почта приходят из
# аргументов, и повторный запуск с новым --domain обязан их обновить, иначе
# Caddy продолжит обслуживать старое имя, а установщик отчитается о новом.
set_env_key() {
    local file=$1 key=$2 value=$3

    if [[ "$(env_value "${file}" "${key}")" == "${value}" ]]; then
        return 0
    fi
    if ! grep -q "^${key}=" "${file}"; then
        ensure_env_key "${file}" "${key}" "${value}"
        return 0
    fi
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] would set ${key}=${value} in ${file}"
        return 0
    fi

    log_info "Updating ${key} in ${file}"
    sed -i "s|^${key}=.*|${key}=${value}|" "${file}"
}

# Write .env from .env.example, substituting the domain and the ACME e-mail.
# A repeated run keeps the existing file: it is the operator's, not ours.
create_env_file() {
    local env_file="${INSTALL_DIR}/.env"

    if [[ -f "${env_file}" && "${REINSTALL}" != "true" ]]; then
        log_info "Keeping existing ${env_file}"
        # Файл может быть старше этого скрипта: без FFS_IMAGE compose не
        # запустится вовсе, без BACKUP_DIR бэкапы легли бы внутрь тома с базой.
        # FFS_IMAGE только добавляется: на работающей установке его значение
        # принадлежит последнему выкату, а не установщику.
        ensure_env_key "${env_file}" FFS_IMAGE "${IMAGE}"
        ensure_env_key "${env_file}" DATABASE_PATH /data/budget.db
        ensure_env_key "${env_file}" BACKUP_DIR /backups
        ensure_env_key "${env_file}" BACKUP_KEEP 30
        set_env_key "${env_file}" DOMAIN "${DOMAIN}"
        set_env_key "${env_file}" ACME_EMAIL "${ACME_EMAIL}"
        return 0
    fi

    log_info "Creating ${env_file}..."

    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] would render ${DEPLOY_DIR}/.env.example with DOMAIN=${DOMAIN}, ACME_EMAIL=${ACME_EMAIL}, FFS_IMAGE=${IMAGE}"
        return 0
    fi

    sed -e "s|^DOMAIN=.*|DOMAIN=${DOMAIN}|" \
        -e "s|^ACME_EMAIL=.*|ACME_EMAIL=${ACME_EMAIL}|" \
        -e "s|^FFS_IMAGE=.*|FFS_IMAGE=${IMAGE}|" \
        "${DEPLOY_DIR}/.env.example" > "${env_file}"

    chmod 600 "${env_file}"
    log_success "Created ${env_file}"
}

set_file_permissions() {
    log_info "Setting permissions..."

    # The container runs as 1000:1000; without this SQLite cannot create the
    # database on the bind mount and Caddy cannot write its access log (D-03).
    run chown -R "${CONTAINER_UID}:${CONTAINER_GID}" \
        "${INSTALL_DIR}/data" "${INSTALL_DIR}/backups" "${INSTALL_DIR}/logs"
    run chmod 700 "${INSTALL_DIR}/data" "${INSTALL_DIR}/backups"

    log_success "Permissions set"
}

# Compose refuses to create its network while the old one holds 172.20.0.0/16
# ("Pool overlaps with other one on this address space").
remove_legacy_network() {
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] docker network rm ${LEGACY_NETWORK}"
        return 0
    fi

    if ! docker network inspect "${LEGACY_NETWORK}" &>/dev/null; then
        return 0
    fi

    if docker network rm "${LEGACY_NETWORK}" 2>/dev/null; then
        log_info "Removed the network of the previous installation: ${LEGACY_NETWORK}"
        return 0
    fi

    # Сеть не удаляется, пока к ней подключены контейнеры, — `up` ниже упадёт
    # на "Pool overlaps", поэтому показываем оператору, что именно её держит.
    log_warning "Could not remove the legacy network ${LEGACY_NETWORK}; containers still attached:"
    docker network inspect -f '{{range .Containers}}  {{.Name}}{{println}}{{end}}' "${LEGACY_NETWORK}" || true
    log_warning "Stop and remove them (docker rm -f <name>), then re-run the installer"
}

deploy_application() {
    log_info "Pulling ${IMAGE}..."

    run cd "${INSTALL_DIR}"
    # Реестр закрытый: без `docker login` pull отвечает про доступ, и сообщение
    # стоит объяснить здесь, а не оставить голым выводом docker.
    if ! run docker compose pull; then
        log_error "Pull failed. The registry is private — log in first:"
        log_error "  docker login ${IMAGE%%/*}"
        exit 1
    fi

    remove_legacy_network

    log_info "Starting services..."
    run docker compose up -d --wait

    reload_caddy

    log_success "Application deployed"
}

# Caddy падает на битом конфиге и на пустом ACME_EMAIL, а restart: unless-stopped
# прячет это от `up -d`: без явной проверки установка отчитывается об успехе,
# пока ingress крутится в перезапусках.
verify_caddy() {
    local state
    state="$(docker inspect --format '{{.State.Status}}' family-budget-caddy 2>/dev/null || echo missing)"
    if [[ "${state}" == "running" ]]; then
        return 0
    fi

    log_error "Caddy is not running (state: ${state}); the API is not reachable over HTTPS"
    docker compose -f "${INSTALL_DIR}/docker-compose.yml" logs --tail 50 caddy
    exit 1
}

verify_installation() {
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[dry-run] would wait for the app container to become healthy"
        return 0
    fi

    log_info "Waiting for the application to become healthy..."

    local attempt=0
    local status=""
    while [[ ${attempt} -lt 30 ]]; do
        status="$(docker inspect --format '{{.State.Health.Status}}' family-budget-app 2>/dev/null || echo missing)"
        if [[ "${status}" == "healthy" ]]; then
            log_success "Application is healthy"
            verify_caddy
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    log_error "Application did not become healthy (last status: ${status})"
    docker compose -f "${INSTALL_DIR}/docker-compose.yml" logs --tail 50
    exit 1
}

show_completion_message() {
    echo ""
    log_success "Family Budget Service installed at ${INSTALL_DIR}"
    echo ""
    log_info "1. Create the family and the first admin (the password is read from stdin):"
    cat <<EOF

   cd ${INSTALL_DIR} && printf 'YourPassword1!\\n' | docker compose exec -T app \\
       /app/family-budget-service setup --family 'Family' --currency RUB \\
       --timezone Europe/Moscow --email you@example.com \\
       --first-name Name --last-name Surname --password-stdin

EOF
    log_info "2. Add the daily backup to the host crontab (retention: BACKUP_KEEP in .env):"
    cat <<EOF

   0 3 * * * cd ${INSTALL_DIR} && docker compose run --rm --no-deps -T app backup

EOF
    log_info "Second user: POST /api/v1/users as the admin. API: https://${DOMAIN}/api/v1"
    log_info "Logs: docker compose -f ${INSTALL_DIR}/docker-compose.yml logs -f"
    log_info "Database: ${INSTALL_DIR}/data/budget.db, backups: ${INSTALL_DIR}/backups/"
    echo ""
}

main() {
    parse_args "$@"

    if [[ "${DRY_RUN}" != "true" ]]; then
        exec 1> >(tee -a "${LOG_FILE}")
        exec 2>&1
    fi

    echo ""
    log_info "Family Budget Service - installation"
    echo ""

    check_root
    detect_os

    if [[ "${DRY_RUN}" == "true" ]]; then
        log_warning "--dry-run: system checks are skipped, nothing is written"
    else
        check_system_requirements
        check_ports
    fi

    prompt_configuration

    run install_docker
    run setup_firewall
    create_directories
    check_deploy_dir
    copy_deploy_files
    create_env_file
    set_file_permissions
    deploy_application
    verify_installation

    show_completion_message
}

main "$@"
