#!/bin/bash
# Family Budget Service - Health Check Script
# This script checks if the application is responding correctly

set -euo pipefail

# Configuration from environment or defaults.
# Порт 8080 наружу не публикуется (снаружи слушает только Caddy), поэтому
# по умолчанию опрашиваем домен; для DOMAIN=localhost нужен HEALTH_URL с http.
DOMAIN="${DOMAIN:-localhost}"
HEALTH_URL="${HEALTH_URL:-https://${DOMAIN}/health}"
TIMEOUT="${HEALTH_TIMEOUT:-5}"
RETRIES="${HEALTH_RETRIES:-3}"
RETRY_DELAY="${HEALTH_RETRY_DELAY:-2}"

# Check health endpoint with retries
check_health() {
    local attempt=1
    local http_code
    
    while [[ $attempt -le $RETRIES ]]; do
        http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT}" "${HEALTH_URL}" 2>/dev/null || echo "000")
        
        if [[ "${http_code}" == "200" ]]; then
            echo "Health check PASSED (HTTP ${http_code})"
            return 0
        fi
        
        if [[ $attempt -lt $RETRIES ]]; then
            echo "Health check failed (HTTP ${http_code}), retrying in ${RETRY_DELAY}s... (attempt $attempt/$RETRIES)"
            sleep "${RETRY_DELAY}"
        fi
        
        attempt=$((attempt + 1))
    done
    
    echo "Health check FAILED after ${RETRIES} attempts (HTTP ${http_code})"
    return 1
}

# Run health check
if check_health; then
    exit 0
else
    exit 1
fi
