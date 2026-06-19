package pro.d11l.fitcoach.data

import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import pro.d11l.fitcoach.core.db.ExerciseEntity
import pro.d11l.fitcoach.core.db.ExerciseWithSets
import pro.d11l.fitcoach.core.db.SessionDao
import pro.d11l.fitcoach.core.db.SessionEntity
import pro.d11l.fitcoach.core.db.SessionWithExercises
import pro.d11l.fitcoach.core.db.SetEntity
import pro.d11l.fitcoach.core.network.AgingBlockDto
import pro.d11l.fitcoach.core.network.ReasoningNoteDto
import pro.d11l.fitcoach.core.network.SafetyFindingDto
import pro.d11l.fitcoach.core.network.SessionDto
import pro.d11l.fitcoach.core.network.SessionExerciseDto
import pro.d11l.fitcoach.core.network.SessionInputsSummaryDto
import pro.d11l.fitcoach.core.network.SetPrescriptionDto

/**
 * A generated session cached on the device (domain type, decoupled from Room).
 * Carries the [clientSessionId] — the stable idempotency key reused by the
 * offline sync queue (E12-PR2) — alongside the renderable session and its
 * lifecycle status.
 */
data class CachedSession(
    val clientSessionId: String,
    val session: SessionDto,
    val status: String,
    val completedAt: String?,
)

/**
 * Offline cache of the current generated session (E5-PR5). Interface so
 * repositories are unit-testable with an in-memory fake; the production impl
 * wraps the Room DAO, normalizing the session into session/exercise/set rows.
 */
interface SessionCache {
    /** Persists [session] as the current plan under [clientSessionId]; replaces any prior cache. */
    suspend fun save(session: SessionDto, clientSessionId: String): CachedSession

    /** The most recently generated session, or null if none is cached. */
    suspend fun latest(): CachedSession?

    /** The cached session flattened into ordered player steps (E6); null if none cached. */
    suspend fun loadPlan(): SessionPlan?

    /** Persists logged actuals for one set, offline (E6-PR2). */
    suspend fun logSet(setId: Long, logged: LoggedSetState)

    /** Marks the cached session completed (E6-PR5). */
    suspend fun markCompleted(sessionId: String, completedAt: String)

    suspend fun clear()
}

/** Room-backed [SessionCache]. Mapping is delegated to the pure functions below. */
class RoomSessionCache(
    private val dao: SessionDao,
    private val json: Json,
) : SessionCache {

    override suspend fun save(session: SessionDto, clientSessionId: String): CachedSession {
        val (entity, exercises) = session.toCacheGraph(clientSessionId, json)
        dao.replace(entity, exercises)
        return CachedSession(clientSessionId, session, entity.status, entity.completedAt)
    }

    override suspend fun latest(): CachedSession? = dao.latest()?.toCachedSession(json)

    override suspend fun loadPlan(): SessionPlan? = dao.latest()?.toSessionPlan(json)

    override suspend fun logSet(setId: Long, logged: LoggedSetState) {
        dao.updateSetLog(
            setId = setId,
            repsDone = logged.repsDone,
            loadKgDone = logged.loadKgDone,
            rpeActual = logged.rpeActual,
            durationDoneSec = logged.durationDoneSec,
            skipped = logged.skipped,
            completed = logged.completed,
        )
    }

    override suspend fun markCompleted(sessionId: String, completedAt: String) {
        dao.updateSessionStatus(sessionId, SessionEntity.STATUS_COMPLETED, completedAt)
    }

    override suspend fun clear() = dao.clearSessions()
}

// ---- Pure DTO <-> entity mapping (no Room runtime; JVM unit-testable) ----

private val reasoningListSerializer = ListSerializer(ReasoningNoteDto.serializer())
private val safetyListSerializer = ListSerializer(SafetyFindingDto.serializer())
private val stringListSerializer = ListSerializer(String.serializer())

/** Decomposes a generated session into its normalized cache graph. */
fun SessionDto.toCacheGraph(
    clientSessionId: String,
    json: Json,
): Pair<SessionEntity, List<ExerciseWithSets>> {
    val entity = SessionEntity(
        sessionId = id,
        clientSessionId = clientSessionId,
        generatedAt = generatedAt,
        schemaVersion = schemaVersion,
        model = model,
        disclaimer = disclaimer,
        inputsSummaryJson = inputsSummary?.let { json.encodeToString(SessionInputsSummaryDto.serializer(), it) },
        reasoningJson = json.encodeToString(reasoningListSerializer, reasoning),
        safetyFindingsJson = json.encodeToString(safetyListSerializer, safetyFindings),
        agingEmphasesJson = json.encodeToString(stringListSerializer, agingBlock.emphases),
    )
    val exercises = buildList {
        addAll(warmup.toExerciseGraph(id, ExerciseEntity.BLOCK_WARMUP))
        addAll(mainWork.toExerciseGraph(id, ExerciseEntity.BLOCK_MAIN))
        addAll(accessory.toExerciseGraph(id, ExerciseEntity.BLOCK_ACCESSORY))
        addAll(agingBlock.items.toExerciseGraph(id, ExerciseEntity.BLOCK_AGING))
    }
    return entity to exercises
}

private fun List<SessionExerciseDto>.toExerciseGraph(
    sessionId: String,
    blockType: String,
): List<ExerciseWithSets> = mapIndexed { index, ex ->
    ExerciseWithSets(
        exercise = ExerciseEntity(
            sessionId = sessionId,
            blockType = blockType,
            orderIndex = index,
            name = ex.name,
            movement = ex.movement,
            region = ex.region,
            notes = ex.notes,
        ),
        sets = ex.sets.mapIndexed { setIndex, set ->
            SetEntity(
                exerciseId = 0,
                orderIndex = setIndex,
                type = set.type,
                reps = set.reps,
                loadKg = set.loadKg,
                rpeTarget = set.rpeTarget,
                durationSec = set.durationSec,
                restSec = set.restSec,
            )
        },
    )
}

/** Reconstructs the renderable [SessionDto] from a cached graph. */
fun SessionWithExercises.toSessionDto(json: Json): SessionDto {
    val byBlock = exercises.sortedBy { it.exercise.orderIndex }
        .groupBy { it.exercise.blockType }
    fun block(type: String): List<SessionExerciseDto> =
        byBlock[type].orEmpty().map { it.toExerciseDto() }

    return SessionDto(
        id = session.sessionId,
        generatedAt = session.generatedAt,
        schemaVersion = session.schemaVersion,
        model = session.model,
        inputsSummary = session.inputsSummaryJson
            ?.let { json.decodeFromString(SessionInputsSummaryDto.serializer(), it) },
        warmup = block(ExerciseEntity.BLOCK_WARMUP),
        mainWork = block(ExerciseEntity.BLOCK_MAIN),
        accessory = block(ExerciseEntity.BLOCK_ACCESSORY),
        agingBlock = AgingBlockDto(
            emphases = json.decodeFromString(stringListSerializer, session.agingEmphasesJson),
            items = block(ExerciseEntity.BLOCK_AGING),
        ),
        reasoning = json.decodeFromString(reasoningListSerializer, session.reasoningJson),
        safetyFindings = json.decodeFromString(safetyListSerializer, session.safetyFindingsJson),
        disclaimer = session.disclaimer,
    )
}

fun SessionWithExercises.toCachedSession(json: Json): CachedSession = CachedSession(
    clientSessionId = session.clientSessionId,
    session = toSessionDto(json),
    status = session.status,
    completedAt = session.completedAt,
)

private fun ExerciseWithSets.toExerciseDto(): SessionExerciseDto = SessionExerciseDto(
    name = exercise.name,
    movement = exercise.movement,
    region = exercise.region,
    notes = exercise.notes,
    sets = sets.sortedBy { it.orderIndex }.map {
        SetPrescriptionDto(
            type = it.type,
            reps = it.reps,
            loadKg = it.loadKg,
            rpeTarget = it.rpeTarget,
            durationSec = it.durationSec,
            restSec = it.restSec,
        )
    },
)
