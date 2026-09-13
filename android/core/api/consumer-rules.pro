# Модель ответа, у которой приложение не читает ни одного поля (CategoryOk после
# createCategory), R8 удаляет целиком и подставляет в generic-сигнатуру suspend-метода
# Retrofit `Object`: конвертер для него не находится, и первый же вызов роняет приложение.
-keep,allowobfuscation @kotlinx.serialization.Serializable class tech.shatrov.familyfinances.core.api.**
