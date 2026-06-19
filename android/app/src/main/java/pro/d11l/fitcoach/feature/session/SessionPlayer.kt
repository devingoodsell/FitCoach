package pro.d11l.fitcoach.feature.session

import pro.d11l.fitcoach.core.network.SetPrescriptionDto
import pro.d11l.fitcoach.data.LoggedSetState

/** Editable inputs for the current set; text-backed for direct TextField binding. */
data class SetDraft(
    val reps: String = "",
    val loadKg: String = "",
    val durationSec: String = "",
)

/**
 * Pure in-session helpers (E6): seed editable inputs from the prescription, turn
 * a draft into a logged result (defaulting any blank field to the prescribed
 * target), and format a prescription for display. No Android types — JVM-tested.
 */
object SessionPlayer {

    /** Seeds the editable draft from the prescribed targets (E6-PR2 defaults). */
    fun draftFor(p: SetPrescriptionDto): SetDraft = SetDraft(
        reps = p.reps?.toString() ?: "",
        loadKg = p.loadKg?.let(::trimZeros) ?: "",
        durationSec = p.durationSec?.toString() ?: "",
    )

    /** Logs the set as completed, defaulting any blank/invalid field to the target. */
    fun logFrom(p: SetPrescriptionDto, draft: SetDraft): LoggedSetState = LoggedSetState(
        repsDone = draft.reps.trim().toIntOrNull() ?: p.reps,
        loadKgDone = draft.loadKg.trim().toDoubleOrNull() ?: p.loadKg,
        durationDoneSec = draft.durationSec.trim().toIntOrNull() ?: p.durationSec,
        skipped = false,
        completed = true,
    )

    /** A skipped set: recorded, not completed (E6-PR5 partial sessions). */
    fun skipped(): LoggedSetState = LoggedSetState(skipped = true, completed = false)
}

/** Renders a set prescription as a single human line. Loads are shown in kg. */
internal fun formatPrescription(set: SetPrescriptionDto): String {
    val parts = mutableListOf<String>()
    when (set.type) {
        "time" -> set.durationSec?.let { parts.add("${it}s") }
        "distance" -> parts.add("distance")
        else -> set.reps?.let { parts.add("$it reps") }
    }
    set.loadKg?.let { parts.add("${trimZeros(it)} kg") }
    set.rpeTarget?.let { parts.add("RPE ${trimZeros(it)}") }
    set.restSec?.let { parts.add("rest ${it}s") }
    return parts.joinToString(" · ")
}

internal fun trimZeros(v: Double): String =
    if (v == v.toLong().toDouble()) v.toLong().toString() else v.toString()
