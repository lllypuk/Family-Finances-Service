package tech.shatrov.familyfinances.core.api.net

import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerializationException
import kotlinx.serialization.descriptors.PrimitiveKind
import kotlinx.serialization.descriptors.PrimitiveSerialDescriptor
import kotlinx.serialization.descriptors.SerialDescriptor
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder
import kotlinx.serialization.modules.SerializersModule
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.format.DateTimeParseException
import java.util.UUID

// DateTimeParseException и IllegalArgumentException из UUID.fromString — не наследники
// SerializationException, и ApiClient их не ловит: без обёртки кривая дата в ответе роняет
// приложение вместо ApiFailure.Malformed.
private fun <T> parsed(
    raw: String,
    parse: (String) -> T,
): T = try {
    parse(raw)
} catch (e: IllegalArgumentException) {
    throw SerializationException("значение не разобрано: $raw", e)
} catch (e: DateTimeParseException) {
    throw SerializationException("значение не разобрано: $raw", e)
}

/** Календарная дата `YYYY-MM-DD`: ни времени, ни зоны в API нет (A-06). */
internal object LocalDateSerializer : KSerializer<LocalDate> {
    override val descriptor: SerialDescriptor =
        PrimitiveSerialDescriptor("java.time.LocalDate", PrimitiveKind.STRING)

    override fun serialize(
        encoder: Encoder,
        value: LocalDate,
    ) = encoder.encodeString(value.toString())

    override fun deserialize(decoder: Decoder): LocalDate = parsed(decoder.decodeString(), LocalDate::parse)
}

/** RFC3339 UTC — `created_at`, `expires_at`, `meta.timestamp`. */
internal object OffsetDateTimeSerializer : KSerializer<OffsetDateTime> {
    override val descriptor: SerialDescriptor =
        PrimitiveSerialDescriptor("java.time.OffsetDateTime", PrimitiveKind.STRING)

    override fun serialize(
        encoder: Encoder,
        value: OffsetDateTime,
    ) = encoder.encodeString(value.toString())

    override fun deserialize(decoder: Decoder): OffsetDateTime = parsed(decoder.decodeString(), OffsetDateTime::parse)
}

internal object UuidSerializer : KSerializer<UUID> {
    override val descriptor: SerialDescriptor =
        PrimitiveSerialDescriptor("java.util.UUID", PrimitiveKind.STRING)

    override fun serialize(
        encoder: Encoder,
        value: UUID,
    ) = encoder.encodeString(value.toString())

    override fun deserialize(decoder: Decoder): UUID = parsed(decoder.decodeString(), UUID::fromString)
}

// Генератор помечает даты и идентификаторы `@Contextual`; без этого модуля не разбирается ни один
// ответ конверта — `meta.timestamp` есть даже у логина.
val apiSerializersModule: SerializersModule = SerializersModule {
    contextual(LocalDate::class, LocalDateSerializer)
    contextual(OffsetDateTime::class, OffsetDateTimeSerializer)
    contextual(UUID::class, UuidSerializer)
}
