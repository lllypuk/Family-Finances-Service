/* Дополнительная JavaScript логика для семейного бюджета */

document.addEventListener("DOMContentLoaded", function () {
  // Инициализация приложения
  initFlashMessages();
  initFormValidation();
  initDropdowns();
  initProgressBars();

  console.log("Family Budget App initialized");
});

// Управление flash сообщениями
function initFlashMessages() {
  const flashMessages = document.querySelectorAll(".alert");

  flashMessages.forEach(function (alert) {
    const closeBtn = alert.querySelector(".close");

    if (closeBtn) {
      closeBtn.addEventListener("click", function () {
        alert.style.opacity = "0";
        setTimeout(function () {
          alert.remove();
        }, 300);
      });
    }

    // Автоматическое скрытие через 5 секунд для success сообщений
    if (alert.classList.contains("alert-success")) {
      setTimeout(function () {
        if (alert.parentNode) {
          alert.style.opacity = "0";
          setTimeout(function () {
            alert.remove();
          }, 300);
        }
      }, 5000);
    }
  });
}

// Валидация форм в реальном времени
function initFormValidation() {
  const forms = document.querySelectorAll("form[data-validate]");

  forms.forEach(function (form) {
    const inputs = form.querySelectorAll("input, select, textarea");

    inputs.forEach(function (input) {
      input.addEventListener("blur", function () {
        validateField(input);
      });

      input.addEventListener("input", function () {
        clearFieldError(input);
      });
    });

    form.addEventListener("submit", function (e) {
      let isValid = true;

      inputs.forEach(function (input) {
        if (!validateField(input)) {
          isValid = false;
        }
      });

      if (!isValid) {
        e.preventDefault();
        return false;
      }
    });
  });
}

// Валидация отдельного поля
function validateField(field) {
  const value = field.value.trim();
  const fieldName = field.name || field.id;
  let isValid = true;
  let errorMessage = "";

  // Проверка обязательных полей
  if (field.hasAttribute("required") && !value) {
    isValid = false;
    errorMessage = "Это поле обязательно для заполнения";
  }

  // Проверка email
  if (field.type === "email" && value) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(value)) {
      isValid = false;
      errorMessage = "Введите корректный email адрес";
    }
  }

  // Проверка минимальной длины
  const minLength = field.getAttribute("minlength");
  if (minLength && value.length < parseInt(minLength)) {
    isValid = false;
    errorMessage = `Минимальная длина: ${minLength} символов`;
  }

  // Проверка пароля
  if (field.type === "password" && value.length > 0 && value.length < 6) {
    isValid = false;
    errorMessage = "Пароль должен содержать минимум 6 символов";
  }

  // Отображение ошибки
  if (!isValid) {
    showFieldError(field, errorMessage);
  } else {
    clearFieldError(field);
  }

  return isValid;
}

// Показать ошибку поля
function showFieldError(field, message) {
  clearFieldError(field);

  field.style.borderColor = "var(--del)";

  const errorDiv = document.createElement("div");
  errorDiv.className = "form-error";
  errorDiv.textContent = message;
  errorDiv.id = field.name + "-error";

  field.parentNode.appendChild(errorDiv);
}

// Очистить ошибку поля
function clearFieldError(field) {
  field.style.borderColor = "";

  const existingError = document.getElementById(field.name + "-error");
  if (existingError) {
    existingError.remove();
  }
}

// Инициализация выпадающих меню
function initDropdowns() {
  const dropdowns = document.querySelectorAll(".dropdown");

  dropdowns.forEach(function (dropdown) {
    const summary = dropdown.querySelector("summary");
    const menu = dropdown.querySelector("ul");

    if (summary && menu) {
      summary.addEventListener("click", function (e) {
        e.preventDefault();
        dropdown.classList.toggle("open");
      });

      // Закрытие при клике вне меню
      document.addEventListener("click", function (e) {
        if (!dropdown.contains(e.target)) {
          dropdown.classList.remove("open");
        }
      });
    }
  });
}

// Анимация прогресс-баров
function initProgressBars() {
  const progressBars = document.querySelectorAll(".progress-bar");

  progressBars.forEach(function (bar) {
    const targetWidth = bar.getAttribute("data-width") || "0%";

    // Анимация при загрузке
    setTimeout(function () {
      bar.style.width = targetWidth;
    }, 100);
  });
}

// HTMX события
document.body.addEventListener("htmx:configRequest", function (evt) {
  // Добавление CSRF токена ко всем HTMX запросам
  const csrfToken = document.querySelector('meta[name="csrf-token"]');
  if (csrfToken) {
    evt.detail.headers["X-CSRF-Token"] = csrfToken.getAttribute("content");
  }
});

document.body.addEventListener("htmx:beforeRequest", function (evt) {
  // Показать индикатор загрузки
  const target = evt.target;
  target.style.opacity = "0.7";
  target.style.pointerEvents = "none";
});

document.body.addEventListener("htmx:afterRequest", function (evt) {
  // Скрыть индикатор загрузки
  const target = evt.target;
  target.style.opacity = "";
  target.style.pointerEvents = "";

  // Переинициализация компонентов после HTMX обновления
  initProgressBars();
  initFlashMessages();
});

document.body.addEventListener("htmx:responseError", function (evt) {
  // Обработка ошибок HTMX
  console.error("HTMX Request failed:", evt.detail.xhr.status);

  // Показать пользователю сообщение об ошибке
  showNotification("Произошла ошибка при загрузке данных", "error");
});

// Утилиты
function showNotification(message, type = "info") {
  const notification = document.createElement("div");
  notification.className = `alert alert-${type} fade-in`;

  // Создаем текстовый элемент для сообщения (безопасно)
  const messageSpan = document.createElement("span");
  messageSpan.textContent = message;

  // Создаем кнопку закрытия
  const closeBtn = document.createElement("button");
  closeBtn.type = "button";
  closeBtn.className = "close";

  const closeSpan = document.createElement("span");
  closeSpan.innerHTML = "&times;";
  closeBtn.appendChild(closeSpan);

  // Собираем уведомление
  notification.appendChild(messageSpan);
  notification.appendChild(closeBtn);

  // Добавить в начало контейнера
  const container = document.querySelector(".container") || document.body;
  container.insertBefore(notification, container.firstChild);

  // Добавить обработчик закрытия
  closeBtn.addEventListener("click", function () {
    notification.remove();
  });

  // Автоматическое удаление через 5 секунд
  setTimeout(function () {
    if (notification.parentNode) {
      notification.remove();
    }
  }, 5000);
}

function formatCurrency(amount, currency = "RUB") {
  const formatter = new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: currency,
    minimumFractionDigits: 2,
  });

  return formatter.format(amount);
}

function formatDate(date) {
  if (typeof date === "string") {
    date = new Date(date);
  }

  const formatter = new Intl.DateTimeFormat("ru-RU", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  return formatter.format(date);
}

// Экспорт функций для использования в других скриптах
window.FamilyBudget = {
  showNotification: showNotification,
  formatCurrency: formatCurrency,
  formatDate: formatDate,
  validateField: validateField,
};
