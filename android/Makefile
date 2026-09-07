# Единая точка входа для разработчика. `make help` — список целей.
#
# Подпись APK: один и тот же keystore на ноутбуке и в CI, иначе обновление поверх
# требует сносить приложение. Путь и пароль берутся из окружения; локальные значения
# по умолчанию лежат вне репозитория, в ~/.android.
FFS_KEYSTORE_PATH ?= $(HOME)/.android/ffs-installer.jks
FFS_KEYSTORE_PASSWORD ?= $(shell cat $(HOME)/.android/ffs-installer.pass 2>/dev/null)
export FFS_KEYSTORE_PATH
export FFS_KEYSTORE_PASSWORD

GRADLE ?= ./gradlew

.PHONY: help
help: ## Показать список целей
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

.PHONY: compile
compile: ## Скомпилировать отладочную сборку
	$(GRADLE) assembleDebug

.PHONY: test
test: ## Прогнать unit-тесты (Robolectric)
	$(GRADLE) testDebugUnitTest

.PHONY: lint
lint: ## Android Lint
	$(GRADLE) lintDebug

.PHONY: fmt
fmt: ## Переформатировать по ktlint
	$(GRADLE) ktlintFormat

.PHONY: fmt-check
fmt-check: ## Проверить формат по ktlint
	$(GRADLE) ktlintCheck

# Генерации здесь намеренно нет: она тянет артефакты из сети, а `check` обязан работать
# офлайн. Свежесть сгенерированного клиента проверяет отдельная цель `api-check`.
.PHONY: check
check: fmt-check test lint ## Формат, тесты, линт

.PHONY: apk
apk: ## Собрать релизный APK (нужен keystore)
	$(GRADLE) assembleRelease
	@echo "APK: app/build/outputs/apk/release/"

# Генерация вне `check`: она ходит в сеть за генератором и артефактами.
.PHONY: api-gen
api-gen: ## Перегенерировать клиент из docs/api/openapi.yaml (нужна сеть)
	$(GRADLE) :core:api:generateApiClient

# `git diff` новых файлов не видит, поэтому свежесть проверяется по `status --porcelain`.
.PHONY: api-check
api-check: api-gen ## Проверить, что сгенерированный клиент совпадает с контрактом
	@if [ -n "$$(git -C .. status --porcelain -- android/core/api/generated)" ]; then \
		echo "Сгенерированный клиент отстал от docs/api/openapi.yaml — запустите make api-gen и закоммитьте:"; \
		git -C .. status --porcelain -- android/core/api/generated; \
		exit 1; \
	fi

.PHONY: clean
clean: ## Удалить результаты сборки
	$(GRADLE) clean
