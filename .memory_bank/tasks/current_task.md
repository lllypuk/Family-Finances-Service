# Текущая задача: Исправление проблем интерфейса

1. Не работает комбобокс выбора Категории при создании Транзакции

2. http://localhost:8080/categories выдает пустую страницу
  Логи: {"time":"2025-08-31T09:39:16+05:00","level":"ERROR","source":{"function":"family-budget-service/internal/application.NewHTTPServerWithObservability.LoggingMiddleware.func2.1","file":"/home/sasha/GoProjects/Family-Finances-Service/internal/observability/middleware.go","line":85},"msg":"HTTP request failed","request_id":"dcgc3qf8af7d","method":"GET","path":"/categories","remote_addr":"127.0.0.1","user_agent":"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36","status":200,"duration":1733749,"bytes_in":0,"bytes_out":0,"error":"html/template: \"pages/categories/index\" is undefined"}

3. http://localhost:8080/budgets/new выдает пустую страницу
  Логи: {"time":"2025-08-31T09:41:03+05:00","level":"ERROR","source":{"function":"family-budget-service/internal/application.NewHTTPServerWithObservability.LoggingMiddleware.func2.1","file":"/home/sasha/GoProjects/Family-Finances-Service/internal/observability/middleware.go","line":85},"msg":"HTTP request failed","request_id":"dcgc53n3zuqb","method":"GET","path":"/budgets/new","remote_addr":"127.0.0.1","user_agent":"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36","status":200,"duration":2692953,"bytes_in":0,"bytes_out":0,"error":"html/template: \"pages/budgets/new\" is undefined"}
