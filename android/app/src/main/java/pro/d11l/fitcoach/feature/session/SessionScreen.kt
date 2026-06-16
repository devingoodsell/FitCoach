package pro.d11l.fitcoach.feature.session

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.designsystem.MedicalDisclaimer
import pro.d11l.fitcoach.core.network.AgingBlockDto
import pro.d11l.fitcoach.core.network.ReasoningNoteDto
import pro.d11l.fitcoach.core.network.SessionDto
import pro.d11l.fitcoach.core.network.SessionExerciseDto
import pro.d11l.fitcoach.core.network.SessionInputsSummaryDto
import pro.d11l.fitcoach.core.network.SetPrescriptionDto

@Composable
fun SessionScreen(viewModel: SessionViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("Today's workout", style = MaterialTheme.typography.headlineSmall)
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        when {
            state.loading -> Text("Building your session…", style = MaterialTheme.typography.bodyMedium)
            state.session == null ->
                Button(onClick = viewModel::start, modifier = Modifier.fillMaxWidth()) {
                    Text("Start workout")
                }
            else -> SessionContent(state.session!!)
        }

        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())
        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

@Composable
private fun SessionContent(session: SessionDto) {
    session.inputsSummary?.let { InputsSummaryCard(it) }
    ExerciseBlock("Warm-up", session.warmup)
    ExerciseBlock("Main work", session.mainWork)
    ExerciseBlock("Accessory", session.accessory)
    AgingBlockCard(session.agingBlock)
    if (session.reasoning.isNotEmpty()) ReasoningCard(session.reasoning)
}

@Composable
private fun InputsSummaryCard(summary: SessionInputsSummaryDto) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            summary.readinessValue?.let {
                Text("Readiness $it (${summary.readinessConfidence ?: "?"})", style = MaterialTheme.typography.labelLarge)
            }
            summary.locationName?.takeIf { it.isNotBlank() }?.let {
                Text("Location: $it", style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

@Composable
private fun ExerciseBlock(title: String, exercises: List<SessionExerciseDto>) {
    if (exercises.isEmpty()) return
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium)
            exercises.forEach { ExerciseRow(it) }
        }
    }
}

@Composable
private fun AgingBlockCard(block: AgingBlockDto) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text("Healthy-aging block", style = MaterialTheme.typography.titleMedium)
            if (block.emphases.isNotEmpty()) {
                Text("Focus: ${block.emphases.joinToString(", ")}", style = MaterialTheme.typography.labelMedium)
            }
            block.items.forEach { ExerciseRow(it) }
        }
    }
}

@Composable
private fun ExerciseRow(ex: SessionExerciseDto) {
    Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
        Text(ex.name, style = MaterialTheme.typography.bodyLarge)
        ex.sets.forEachIndexed { i, set ->
            Text("Set ${i + 1}: ${formatSet(set)}", style = MaterialTheme.typography.bodySmall)
        }
        ex.notes?.takeIf { it.isNotBlank() }?.let {
            Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun ReasoningCard(notes: List<ReasoningNoteDto>) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Why this session", style = MaterialTheme.typography.titleMedium)
            notes.forEach { Text("• ${it.text}", style = MaterialTheme.typography.bodyMedium) }
        }
    }
}

/** Renders a set prescription as a single human line. Loads are shown in kg
 *  (the canonical unit); unit conversion for display is a later concern. */
private fun formatSet(set: SetPrescriptionDto): String {
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

private fun trimZeros(v: Double): String =
    if (v == v.toLong().toDouble()) v.toLong().toString() else v.toString()

// --- Preview against the published session shape -----------------------------

/** Sample mirroring backend/api/examples/session-sample.json, so the preview and
 *  E6 can build against the shape without a running backend. */
internal fun sampleSession(): SessionDto = SessionDto(
    id = "0192f3a0-1c2d-7e00-9abc-0123456789ab",
    generatedAt = "2026-06-16T13:30:00Z",
    schemaVersion = 1,
    model = "claude-opus-4-8",
    inputsSummary = SessionInputsSummaryDto(
        readinessValue = 72,
        readinessConfidence = "high",
        contraindicationCount = 1,
        locationName = "Home gym",
        agingEmphases = listOf("bone_balance", "joint_tendon"),
    ),
    warmup = listOf(
        SessionExerciseDto("Rower easy spin", "row_erg", "full_body", listOf(
            SetPrescriptionDto(type = "time", durationSec = 180, rpeTarget = 3.0, restSec = 0),
        )),
    ),
    mainWork = listOf(
        SessionExerciseDto("Goblet box squat", "box_squat", "quad", listOf(
            SetPrescriptionDto(type = "reps", reps = 8, loadKg = 20.0, rpeTarget = 6.0, restSec = 120),
            SetPrescriptionDto(type = "reps", reps = 8, loadKg = 24.0, rpeTarget = 7.0, restSec = 120),
        ), notes = "Box keeps load off the knee."),
    ),
    accessory = listOf(
        SessionExerciseDto("Half-kneeling cable row", "row", "back", listOf(
            SetPrescriptionDto(type = "reps", reps = 12, loadKg = 20.0, rpeTarget = 7.0, restSec = 75),
        )),
    ),
    agingBlock = AgingBlockDto(
        emphases = listOf("bone_balance", "joint_tendon"),
        items = listOf(
            SessionExerciseDto("Pogo hops", "low_amplitude_jump", "ankle", listOf(
                SetPrescriptionDto(type = "reps", reps = 15, rpeTarget = 5.0, restSec = 45),
            ), notes = "Bone-loading and tendon stiffness."),
        ),
    ),
    reasoning = listOf(
        ReasoningNoteDto("Held RPE 7-8 with full rest on a strong readiness day.", "intensity"),
        ReasoningNoteDto("At 45, added bone-loading hops and balance work.", "age_aware"),
    ),
    disclaimer = "FitCoach provides general fitness guidance, not medical advice.",
)

@Preview(showBackground = true)
@Composable
private fun SessionContentPreview() {
    Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        SessionContent(sampleSession())
    }
}
